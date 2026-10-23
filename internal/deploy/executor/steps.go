package executor

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/deploy/planner"
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
			_, _ = fmt.Fprintln(logs, "Blue-green is not available yet; using recreate.")
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
func (e *Executor) others(ctx context.Context, j *job, service string) ([]string, error) {
	list, err := e.Docker.ListManaged(ctx, map[string]string{
		domain.LabelApp: j.env.AppName, domain.LabelEnv: j.env.Name, domain.LabelService: service,
	})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, c := range list {
		if c.Name != j.containers[service].Name {
			ids = append(ids, c.ID)
		}
	}
	return ids, nil
}

// recreate stops the old containers of a service, then starts the new one and
// waits until it is healthy. The service is down in between.
func (e *Executor) recreate(ctx context.Context, j *job, service string, logs io.Writer) error {
	cs := j.containers[service]
	old, err := e.others(ctx, j, service)
	if err != nil {
		return err
	}
	for _, id := range old {
		_, _ = fmt.Fprintf(logs, "Stopping old container %.12s\n", id)
		if err := e.Docker.StopContainer(ctx, id, cs.StopTimeout); err != nil {
			return err
		}
		if err := e.Docker.RemoveContainer(ctx, id); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintf(logs, "Starting %s (%s)\n", cs.Name, cs.Image)
	id, err := e.Docker.CreateContainer(ctx, cs)
	if err != nil {
		return err
	}
	if err := e.Docker.StartContainer(ctx, id); err != nil {
		return err
	}
	if err := e.Docker.WaitHealthy(ctx, id, cs); err != nil {
		_, _ = fmt.Fprintln(logs, "Container did not become healthy; its last output:")
		_ = e.Docker.Logs(context.Background(), id, false, logs)
		return err
	}
	_, _ = fmt.Fprintf(logs, "%s is healthy\n", cs.Name)
	return nil
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
	for _, c := range all {
		svc := c.Labels[domain.LabelService]
		_, inSpec := j.spec.Service(svc)
		stale := !inSpec || (!c.Running && !slices.Contains(names, c.Name))
		if !stale {
			continue
		}
		_, _ = fmt.Fprintf(logs, "Removing %s\n", c.Name)
		if err := e.Docker.StopContainer(ctx, c.ID, containerspec.DefaultStop); err != nil {
			return err
		}
		if err := e.Docker.RemoveContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	return nil
}
