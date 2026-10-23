package executor

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/deploy/planner"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// step runs one planned step. Jobs return their exit code.
func (e *Executor) step(ctx context.Context, j *job, st planner.Step, logs io.Writer) (*int, error) {
	envLabels := map[string]string{domain.LabelApp: j.env.AppName, domain.LabelEnv: j.env.Name}
	switch st.Kind {
	case planner.StepEnsureNetwork:
		return nil, e.Docker.EnsureNetwork(ctx, domain.NetworkName(j.env.AppName, j.env.Name), envLabels)
	case planner.StepEnsureVolumes:
		for _, v := range st.Volumes {
			_, _ = fmt.Fprintf(logs, "Volume %s\n", v)
			if err := e.Docker.EnsureVolume(ctx, v, envLabels); err != nil {
				return nil, err
			}
		}
		return nil, nil
	case planner.StepStateful, planner.StepDeploy:
		if st.Strategy == spec.StrategyBlueGreen {
			return nil, e.blueGreen(ctx, j, st.Service, logs)
		}
		return nil, e.recreate(ctx, j, st.Service, logs)
	case planner.StepJob:
		code, err := e.runJob(ctx, j, st.Service, logs)
		return &code, err
	case planner.StepFinish:
		return nil, e.finish(ctx, j, logs)
	default:
		return nil, fmt.Errorf("unknown step %q", st.Kind)
	}
}

// others lists the service's containers that belong to other deployments.
func (e *Executor) others(ctx context.Context, j *job, service string) ([]dockerx.ContainerInfo, error) {
	list, err := e.Docker.ListManaged(ctx, map[string]string{
		domain.LabelApp: j.env.AppName, domain.LabelEnv: j.env.Name, domain.LabelService: service,
	})
	if err != nil {
		return nil, err
	}
	var out []dockerx.ContainerInfo
	for _, c := range list {
		if c.Name != j.containers[service].Name {
			out = append(out, c)
		}
	}
	return out, nil
}

// recreate takes the old containers of a service out of routing, stops them,
// then starts the new one and routes it once healthy. The service is down in
// between.
func (e *Executor) recreate(ctx context.Context, j *job, service string, logs io.Writer) error {
	cs := j.containers[service]
	old, err := e.others(ctx, j, service)
	if err != nil {
		return err
	}
	if err := e.unroute(ctx, j, old); err != nil {
		return err
	}
	for _, c := range old {
		_, _ = fmt.Fprintf(logs, "Stopping old container %s\n", c.Name)
		if err := e.Docker.StopContainer(ctx, c.ID, cs.StopTimeout); err != nil {
			return err
		}
		if err := e.Docker.RemoveContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	_, err = e.startHealthy(ctx, j, cs, logs)
	return err
}

// blueGreen replaces a service without downtime: the new container starts next
// to the old ones and is routed once healthy, so both serve for a moment; then
// the old ones leave routing, drain, get SIGTERM and are removed. If the new
// container never becomes healthy it is removed and the old ones keep serving.
func (e *Executor) blueGreen(ctx context.Context, j *job, service string, logs io.Writer) error {
	cs := j.containers[service]
	old, err := e.others(ctx, j, service)
	if err != nil {
		return err
	}
	if len(old) > 0 {
		_, _ = fmt.Fprintf(logs, "Blue-green: %s replaces %s\n", cs.Name, names(old))
	}
	if _, err := e.startHealthy(ctx, j, cs, logs); err != nil {
		if len(old) > 0 {
			_, _ = fmt.Fprintf(logs, "Aborted: %s keeps serving\n", names(old))
		}
		return err
	}
	if len(old) == 0 {
		return nil
	}
	_, _ = fmt.Fprintf(logs, "Both versions serve; taking %s out of routing\n", names(old))
	if err := e.unroute(ctx, j, old); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(logs, "Draining for %s\n", cs.Drain)
	if err := e.wait(ctx, cs.Drain); err != nil {
		return err
	}
	for _, c := range old {
		_, _ = fmt.Fprintf(logs, "Stopping %s (grace %s)\n", c.Name, cs.StopTimeout)
		if err := e.Docker.StopContainer(ctx, c.ID, cs.StopTimeout); err != nil {
			return err
		}
		if err := e.Docker.RemoveContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	return nil
}

func names(cs []dockerx.ContainerInfo) string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return strings.Join(out, ", ")
}

// startHealthy creates and starts a container, waits until it is healthy and
// only then routes it. A container that never becomes healthy is removed, so
// it never receives traffic.
func (e *Executor) startHealthy(ctx context.Context, j *job, cs containerspec.Spec, logs io.Writer) (string, error) {
	_, _ = fmt.Fprintf(logs, "Starting %s (%s)\n", cs.Name, cs.Image)
	j.excluded[cs.Name] = true
	id, err := e.Docker.CreateContainer(ctx, cs)
	if err != nil {
		return "", err
	}
	if err := e.Docker.StartContainer(ctx, id); err != nil {
		_ = e.Docker.RemoveContainer(context.Background(), id)
		return "", err
	}
	if err := e.Docker.WaitHealthy(ctx, id, cs); err != nil {
		_, _ = fmt.Fprintln(logs, "Container did not become healthy; its last output:")
		_ = e.Docker.Logs(context.Background(), id, false, logs)
		_ = e.Docker.RemoveContainer(context.Background(), id)
		return "", err
	}
	delete(j.excluded, cs.Name)
	_, _ = fmt.Fprintf(logs, "%s is healthy and routed\n", cs.Name)
	return id, e.syncRoutes(ctx, j.env, j.spec, j.excluded)
}

// unroute takes containers out of the routes before they are stopped.
func (e *Executor) unroute(ctx context.Context, j *job, containers []dockerx.ContainerInfo) error {
	if len(containers) == 0 {
		return nil
	}
	for _, c := range containers {
		j.excluded[c.Name] = true
	}
	return e.syncRoutes(ctx, j.env, j.spec, j.excluded)
}

// runJob runs a job container to completion, streaming its output. A non-zero
// exit code fails the step. The container is removed afterwards; its output is
// kept in the deployment log.
func (e *Executor) runJob(ctx context.Context, j *job, service string, logs io.Writer) (int, error) {
	cs := j.containers[service]
	id, err := e.Docker.CreateContainer(ctx, cs)
	if err != nil {
		return -1, err
	}
	defer func() { _ = e.Docker.RemoveContainer(context.Background(), id) }()
	if err := e.Docker.StartContainer(ctx, id); err != nil {
		return -1, err
	}
	logsDone := make(chan struct{})
	go func() {
		defer close(logsDone)
		_ = e.Docker.Logs(ctx, id, true, logs)
	}()
	code, err := e.Docker.WaitExit(ctx, id)
	<-logsDone
	if err != nil {
		return code, err
	}
	_, _ = fmt.Fprintf(logs, "Job %s exited with code %d\n", service, code)
	if code != 0 {
		return code, fmt.Errorf("job %s exited with code %d", service, code)
	}
	return code, nil
}

// finish removes containers of services that are no longer in the spec, and
// leftovers of previous deployments of current services that are not running.
func (e *Executor) finish(ctx context.Context, j *job, logs io.Writer) error {
	all, err := e.Docker.ListManaged(ctx, map[string]string{domain.LabelApp: j.env.AppName, domain.LabelEnv: j.env.Name})
	if err != nil {
		return err
	}
	names := make([]string, 0, len(j.containers))
	for _, cs := range j.containers {
		names = append(names, cs.Name)
	}
	var stale []dockerx.ContainerInfo
	for _, c := range all {
		_, inSpec := j.spec.Service(c.Labels[domain.LabelService])
		if !inSpec || (!c.Running && !slices.Contains(names, c.Name)) {
			stale = append(stale, c)
		}
	}
	if err := e.unroute(ctx, j, stale); err != nil {
		return err
	}
	for _, c := range stale {
		_, _ = fmt.Fprintf(logs, "Removing %s\n", c.Name)
		if err := e.Docker.StopContainer(ctx, c.ID, spec.DefaultStopGracePeriod); err != nil {
			return err
		}
		if err := e.Docker.RemoveContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	return nil
}
