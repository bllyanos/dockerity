package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/deploy/planner"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/routing"
	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/spec"
)

// Executor runs one deployment at a time per call; the environment lock in
// the database guarantees a single deployment per environment.
type Executor struct {
	Store   Store
	Envs    Envs
	Specs   Specs
	Builds  BuildImages
	Secrets Secrets
	Docker  Docker
	Logs    buildlog.Store
	Hub     *buildlog.Hub
	Routing routing.Options
	// Routes, when set, receives each environment's Traefik routes.
	Routes RouteFiles
	// AllEnvs, when set, lets RebuildAllRoutes find every environment.
	AllEnvs AllEnvs
	Logger  *slog.Logger
	Now     func() time.Time
}

// job is everything resolved for one deployment.
type job struct {
	dep        domain.Deployment
	env        domain.Environment
	spec       spec.Spec
	containers map[string]containerspec.Spec
	steps      []domain.DeploymentStep
	plan       planner.Plan
	// excluded holds containers that must not receive traffic yet (or anymore).
	excluded map[string]bool
}

// Run executes a pending deployment to a terminal status.
func (e *Executor) Run(ctx context.Context, id int64) {
	sink, err := e.Logs.OpenSink(id, e.Hub)
	if err != nil {
		e.fail(id, fmt.Errorf("open deployment log: %w", err))
		return
	}
	defer func() { _ = sink.Close() }()

	j, err := e.prepare(ctx, id, sink)
	if err != nil {
		_, _ = fmt.Fprintf(sink, "Deployment failed before it started: %s\n", err)
		e.fail(id, err)
		return
	}
	if err := e.Store.StartDeployment(ctx, id, e.Now().UTC()); err != nil {
		e.Logger.Error("start deployment", "deployment", id, "error", err)
		return
	}
	if err := e.execute(ctx, j, sink); err != nil {
		_, _ = fmt.Fprintf(sink, "\nDeployment failed: %s\n", err)
		_ = e.Store.SkipPendingSteps(context.Background(), id)
		e.fail(id, err)
		return
	}
	_, _ = fmt.Fprintln(sink, "\nDeployment succeeded.")
	if err := e.Store.FinishDeployment(context.Background(), id, domain.DeploymentSucceeded, "", e.Now().UTC()); err != nil {
		e.Logger.Error("finish deployment", "deployment", id, "error", err)
	}
}

func (e *Executor) fail(id int64, cause error) {
	if err := e.Store.FinishDeployment(context.Background(), id, domain.DeploymentFailed, cause.Error(), e.Now().UTC()); err != nil {
		e.Logger.Error("fail deployment", "deployment", id, "error", err)
	}
}

// prepare loads the inputs, builds container specs and plans the steps.
func (e *Executor) prepare(ctx context.Context, id int64, logs io.Writer) (*job, error) {
	dep, err := e.Store.GetDeployment(ctx, id)
	if err != nil {
		return nil, err
	}
	env, err := e.Envs.Get(ctx, dep.EnvID)
	if err != nil {
		return nil, err
	}
	sv, err := e.Specs.Get(ctx, env.AppID, dep.SpecVersion)
	if err != nil {
		return nil, err
	}
	s, err := spec.Load([]byte(sv.YAML))
	if err != nil {
		return nil, fmt.Errorf("spec v%d is invalid: %w", sv.Version, err)
	}
	images, err := e.Builds.ListBuildImages(ctx, dep.BuildID)
	if err != nil {
		return nil, err
	}
	buildImages := map[string]string{}
	for _, img := range images {
		buildImages[img.Service] = img.ImageTag
	}
	values := map[string]string{}
	if dep.SecretVersion > 0 {
		if values, err = e.Secrets.Values(ctx, secrets.EnvScope(env.ID), dep.SecretVersion); err != nil {
			return nil, err
		}
	}
	current, err := e.current(ctx, env)
	if err != nil {
		return nil, err
	}

	j := &job{dep: dep, env: env, spec: s, containers: map[string]containerspec.Spec{}, excluded: map[string]bool{}}
	fingerprints := map[string]string{}
	for _, svc := range s.Services {
		image := svc.Image
		if svc.Build != nil {
			image = buildImages[svc.Name] // a missing image is reported by the planner
		}
		cs, err := containerspec.Build(containerspec.Input{
			App: env.AppName, Env: env.Name, DeploymentID: dep.ID,
			Service: svc, Image: image, Secrets: values,
		})
		if err != nil {
			return nil, fmt.Errorf("service %s: %w", svc.Name, err)
		}
		j.containers[svc.Name] = cs
		fingerprints[svc.Name] = cs.Fingerprint()
	}
	j.plan, err = planner.Make(planner.Input{
		App: env.AppName, Env: env.Name, Spec: s, BuildImages: buildImages,
		SecretNames: secrets.SortedKeys(values), Fingerprints: fingerprints, Current: current,
	})
	if err != nil {
		return nil, err
	}
	planned := make([]domain.DeploymentStep, len(j.plan.Steps))
	for i, st := range j.plan.Steps {
		planned[i] = domain.DeploymentStep{Kind: string(st.Kind), Service: st.Service}
	}
	if j.steps, err = e.Store.InsertSteps(ctx, id, planned); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(logs, "Deploying build #%d with spec v%d and secrets v%d to %s/%s (%d steps)\n",
		dep.BuildID, dep.SpecVersion, dep.SecretVersion, env.AppName, env.Name, len(j.steps))
	if j.plan.Downtime() {
		_, _ = fmt.Fprintln(logs, "Warning: this deployment causes downtime for at least one service.")
	}
	return j, nil
}

// current reports what runs now in the environment, by service.
func (e *Executor) current(ctx context.Context, env domain.Environment) (map[string]planner.Running, error) {
	running, err := e.Docker.ListManaged(ctx, map[string]string{domain.LabelApp: env.AppName, domain.LabelEnv: env.Name})
	if err != nil {
		return nil, err
	}
	out := map[string]planner.Running{}
	for _, c := range running {
		if !c.Running {
			continue
		}
		out[c.Labels[domain.LabelService]] = planner.Running{Image: c.Image, Fingerprint: c.Labels[containerspec.LabelFingerprint]}
	}
	return out, nil
}

// execute runs the steps in order and stops at the first failure. Routes are
// rewritten at the end either way, so they always match what runs.
func (e *Executor) execute(ctx context.Context, j *job, logs io.Writer) (err error) {
	defer func() {
		if rerr := e.syncRoutes(context.Background(), j.env, j.spec, j.excluded); rerr != nil {
			err = errors.Join(err, fmt.Errorf("write routes: %w", rerr))
		}
	}()
	for i, st := range j.plan.Steps {
		row := j.steps[i]
		if err := e.Store.StartStep(ctx, row.ID, e.Now().UTC()); err != nil {
			return err
		}
		label := string(st.Kind)
		if st.Service != "" {
			label += " " + st.Service
		}
		_, _ = fmt.Fprintf(logs, "\n==> [%d/%d] %s\n", i+1, len(j.steps), label)
		exit, err := e.step(ctx, j, st, logs)
		status, msg := domain.StepSucceeded, ""
		if err != nil {
			status, msg = domain.StepFailed, err.Error()
		}
		if ferr := e.Store.FinishStep(context.Background(), row.ID, status, exit, msg, e.Now().UTC()); ferr != nil {
			return errors.Join(err, ferr)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return nil
}
