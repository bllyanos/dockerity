// Package reconcile compares Docker with the database on startup and
// periodically (docs/design.md section 6.5). It reports drift and applies the
// safe repairs: rewriting routes from what runs and removing stopped
// leftovers. Missing containers are only reported; a redeploy recreates them.
package reconcile

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/drift"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// DefaultInterval is how often the reconciler runs.
const DefaultInterval = 5 * time.Minute

// Envs reads environments.
type Envs interface {
	Get(ctx context.Context, id int64) (domain.Environment, error)
	ListAllEnvironments(ctx context.Context) ([]domain.Environment, error)
}

// Deployments reads deployments.
type Deployments interface {
	CurrentDeployment(ctx context.Context, envID int64) (domain.Deployment, error)
	ListDeploymentsByStatus(ctx context.Context, status domain.DeploymentStatus) ([]domain.Deployment, error)
}

// Specs reads spec versions.
type Specs interface {
	Get(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error)
}

// Docker lists and removes managed containers.
type Docker interface {
	ListManaged(ctx context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error)
	RemoveContainer(ctx context.Context, id string) error
}

// Routes reads route files.
type Routes interface {
	Routed(app, env string) (map[string]bool, error)
}

// Rebuilder rewrites an environment's routes from what runs.
type Rebuilder interface {
	RebuildRoutes(ctx context.Context, envID int64) error
}

// Reconciler finds and repairs drift.
type Reconciler struct {
	Envs        Envs
	Deployments Deployments
	Specs       Specs
	Docker      Docker
	Routes      Routes
	Rebuilder   Rebuilder
	Logger      *slog.Logger
	Interval    time.Duration
}

// Check reports an environment's drift without repairing it.
func (r *Reconciler) Check(ctx context.Context, envID int64) (drift.Report, error) {
	env, err := r.Envs.Get(ctx, envID)
	if err != nil {
		return drift.Report{}, err
	}
	return r.compute(ctx, env)
}

// Run reconciles on every interval until ctx is cancelled, starting now.
func (r *Reconciler) Run(ctx context.Context) {
	interval := r.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		r.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce reconciles every environment. Environments with a deployment in
// progress are skipped: the executor owns their containers and routes.
func (r *Reconciler) RunOnce(ctx context.Context) {
	envs, err := r.Envs.ListAllEnvironments(ctx)
	if err != nil {
		r.Logger.Error("list environments", "error", err)
		return
	}
	busy, err := r.busy(ctx)
	if err != nil {
		r.Logger.Error("list active deployments", "error", err)
		return
	}
	for _, env := range envs {
		if busy[env.ID] {
			continue
		}
		if err := r.reconcile(ctx, env); err != nil {
			r.Logger.Error("reconcile environment", "env", env.AppName+"/"+env.Name, "error", err)
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context, env domain.Environment) error {
	rep, err := r.compute(ctx, env)
	if err != nil {
		return err
	}
	name := env.AppName + "/" + env.Name
	for _, it := range rep.Items {
		r.Logger.Warn("drift", "env", name, "kind", it.Kind, "service", it.Service, "container", it.Container, "detail", it.Detail)
	}
	var errs []error
	for _, id := range rep.Remove {
		if err := r.Docker.RemoveContainer(ctx, id); err != nil {
			errs = append(errs, err)
			continue
		}
		r.Logger.Info("removed stopped leftover container", "env", name, "container", id)
	}
	if rep.RewriteRoutes {
		if err := r.Rebuilder.RebuildRoutes(ctx, env.ID); err != nil {
			errs = append(errs, err)
		} else {
			r.Logger.Info("rewrote routes from running containers", "env", name)
		}
	}
	return errors.Join(errs...)
}

// busy returns the environments with a pending or running deployment.
func (r *Reconciler) busy(ctx context.Context) (map[int64]bool, error) {
	out := map[int64]bool{}
	for _, st := range []domain.DeploymentStatus{domain.DeploymentPending, domain.DeploymentRunning} {
		list, err := r.Deployments.ListDeploymentsByStatus(ctx, st)
		if err != nil {
			return nil, err
		}
		for _, d := range list {
			out[d.EnvID] = true
		}
	}
	return out, nil
}

func (r *Reconciler) compute(ctx context.Context, env domain.Environment) (drift.Report, error) {
	in := drift.Input{Services: map[string]drift.ServiceKind{}}
	cur, err := r.Deployments.CurrentDeployment(ctx, env.ID)
	switch {
	case err == nil:
		in.Current = cur.ID
		sv, err := r.Specs.Get(ctx, env.AppID, cur.SpecVersion)
		if err != nil {
			return drift.Report{}, err
		}
		s, err := spec.Load([]byte(sv.YAML))
		if err != nil {
			return drift.Report{}, err
		}
		for _, svc := range s.Services {
			if svc.Kind != spec.KindJob {
				in.Services[svc.Name] = drift.ServiceKind(svc.Kind)
			}
		}
	case !errors.Is(err, domain.ErrNotFound):
		return drift.Report{}, err
	}
	list, err := r.Docker.ListManaged(ctx, map[string]string{domain.LabelApp: env.AppName, domain.LabelEnv: env.Name})
	if err != nil {
		return drift.Report{}, err
	}
	for _, c := range list {
		if domain.IsSystem(c.Labels) {
			continue
		}
		in.Containers = append(in.Containers, drift.Container{
			ID: c.ID, Name: c.Name, Service: c.Labels[domain.LabelService], Deployment: c.Labels[domain.LabelDeployment], Running: c.Running,
		})
	}
	if in.Routed, err = r.Routes.Routed(env.AppName, env.Name); err != nil {
		return drift.Report{}, err
	}
	return drift.Compute(in), nil
}
