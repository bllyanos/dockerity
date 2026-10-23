// Package deployments holds the use cases for starting and reading deployments.
package deployments

import (
	"context"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/executor"
	"github.com/bllyanos/dockerity/internal/deploy/warning"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// Store persists deployments.
type Store interface {
	CreateDeployment(ctx context.Context, d domain.Deployment) (domain.Deployment, error)
	GetDeployment(ctx context.Context, id int64) (domain.Deployment, error)
	ListDeployments(ctx context.Context, envID int64, limit int) ([]domain.Deployment, error)
	ListSteps(ctx context.Context, deploymentID int64) ([]domain.DeploymentStep, error)
}

// Runner executes deployments in the background.
type Runner interface {
	Submit(id int64)
}

// Service starts and reads deployments.
type Service struct {
	Store Store
	Envs  interface {
		Get(ctx context.Context, id int64) (domain.Environment, error)
	}
	Builds interface {
		GetBuild(ctx context.Context, id int64) (domain.Build, error)
	}
	Specs interface {
		Latest(ctx context.Context, appID int64) (domain.AppSpecVersion, error)
	}
	Secrets interface {
		Latest(ctx context.Context, scope secrets.Scope) (secrets.Version, error)
	}
	Runner Runner
	// Previewer plans a deployment without running it.
	Previewer interface {
		Preview(ctx context.Context, d domain.Deployment) (executor.Preview, error)
	}
	// Backups tells when volumes were last backed up, for the warning.
	Backups interface {
		LatestBackup(ctx context.Context, volume string) (domain.Backup, error)
	}
	Now func() time.Time
}

// ListLimit is how many deployments List returns.
const ListLimit = 50

// Start deploys a succeeded build of the environment's app with the app's
// latest spec and the environment's latest secrets.
func (s *Service) Start(ctx context.Context, envID, buildID int64, actor *int64) (domain.Deployment, error) {
	draft, env, err := s.draft(ctx, envID, buildID, actor)
	if err != nil {
		return domain.Deployment{}, err
	}
	d, err := s.Store.CreateDeployment(ctx, draft)
	if errors.Is(err, domain.ErrConflict) {
		return domain.Deployment{}, domain.Conflictf("a deployment is already in progress in %s/%s", env.AppName, env.Name)
	}
	if err != nil {
		return domain.Deployment{}, err
	}
	s.Runner.Submit(d.ID)
	return d, nil
}

// Preview explains what deploying buildID would do, without doing it.
func (s *Service) Preview(ctx context.Context, envID, buildID int64) (warning.Warning, error) {
	draft, env, err := s.draft(ctx, envID, buildID, nil)
	if err != nil {
		return warning.Warning{}, err
	}
	p, err := s.Previewer.Preview(ctx, draft)
	if err != nil {
		return warning.Warning{}, err
	}
	last := map[string]time.Time{}
	for _, svc := range p.Spec.Services {
		for _, v := range svc.Volumes {
			name := domain.VolumeName(env.AppName, env.Name, svc.Name, v.Name)
			b, err := s.Backups.LatestBackup(ctx, name)
			if err == nil {
				last[name] = b.CreatedAt
			} else if !errors.Is(err, domain.ErrNotFound) {
				return warning.Warning{}, err
			}
		}
	}
	return warning.Build(warning.Input{
		App: env.AppName, Env: env.Name, Spec: p.Spec, Plan: p.Plan, Current: p.Current, LastBackups: last,
	}), nil
}

// draft validates a deploy request and returns the deployment it would create.
func (s *Service) draft(ctx context.Context, envID, buildID int64, actor *int64) (domain.Deployment, domain.Environment, error) {
	env, err := s.Envs.Get(ctx, envID)
	if err != nil {
		return domain.Deployment{}, env, err
	}
	b, err := s.Builds.GetBuild(ctx, buildID)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && b.AppID != env.AppID) {
		return domain.Deployment{}, env, domain.Invalid("buildId", "build %d does not belong to app %s", buildID, env.AppName)
	}
	if err != nil {
		return domain.Deployment{}, env, err
	}
	if b.Status != domain.BuildSucceeded {
		return domain.Deployment{}, env, domain.Invalid("buildId", "build %d is %s; only succeeded builds can be deployed", buildID, b.Status)
	}
	sv, err := s.Specs.Latest(ctx, env.AppID)
	if err != nil {
		return domain.Deployment{}, env, err
	}
	secretVersion := 0
	if v, err := s.Secrets.Latest(ctx, secrets.EnvScope(env.ID)); err == nil {
		secretVersion = v.Version
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Deployment{}, env, err
	}

	return domain.Deployment{
		EnvID: env.ID, BuildID: b.ID, SpecVersion: sv.Version, SecretVersion: secretVersion,
		TriggeredBy: actor, CreatedAt: s.Now().UTC(),
	}, env, nil
}

// List returns an environment's recent deployments.
func (s *Service) List(ctx context.Context, envID int64) ([]domain.Deployment, error) {
	if _, err := s.Envs.Get(ctx, envID); err != nil {
		return nil, err
	}
	return s.Store.ListDeployments(ctx, envID, ListLimit)
}

// Get returns a deployment and its steps.
func (s *Service) Get(ctx context.Context, id int64) (domain.Deployment, []domain.DeploymentStep, error) {
	d, err := s.Store.GetDeployment(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return d, nil, domain.NotFoundf("deployment %d not found", id)
	}
	if err != nil {
		return d, nil, err
	}
	steps, err := s.Store.ListSteps(ctx, id)
	return d, steps, err
}
