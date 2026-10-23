package executor

import (
	"context"
	"sync"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Runner executes submitted deployments in the background. Deployments of
// different environments run in parallel; the environment lock in the
// database prevents two on the same environment.
type Runner struct {
	Exec *Executor

	ctx context.Context
	wg  sync.WaitGroup
}

// Start fails deployments left unfinished by a crash and remembers ctx for
// the deployments it runs.
func (r *Runner) Start(ctx context.Context) error {
	r.ctx = ctx
	for _, status := range []domain.DeploymentStatus{domain.DeploymentRunning, domain.DeploymentPending} {
		stale, err := r.Exec.Store.ListDeploymentsByStatus(ctx, status)
		if err != nil {
			return err
		}
		for _, d := range stale {
			_ = r.Exec.Store.SkipPendingSteps(ctx, d.ID)
			if err := r.Exec.Store.FinishDeployment(ctx, d.ID, domain.DeploymentFailed,
				"interrupted: Dockerity restarted during the deployment", r.Exec.Now().UTC()); err != nil {
				return err
			}
			r.Exec.Logger.Warn("marked interrupted deployment as failed", "deployment", d.ID)
		}
	}
	// Containers may have changed while Dockerity was down; make routes match.
	if err := r.Exec.RebuildAllRoutes(ctx); err != nil {
		r.Exec.Logger.Error("rebuild routes", "error", err)
	}
	return nil
}

// Submit runs a pending deployment in the background.
func (r *Runner) Submit(id int64) {
	r.wg.Go(func() { r.Exec.Run(r.ctx, id) })
}

// Wait blocks until submitted deployments finished.
func (r *Runner) Wait() { r.wg.Wait() }
