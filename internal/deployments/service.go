// Package deployments holds the use cases for starting and reading deployments.
package deployments

import (
	"context"
	"errors"
	"time"

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
	Now    func() time.Time
}

// ListLimit is how many deployments List returns.
const ListLimit = 50

// Start deploys a succeeded build of the environment's app with the app's
// latest spec and the environment's latest secrets.
func (s *Service) Start(ctx context.Context, envID, buildID int64, actor *int64) (domain.Deployment, error) {
	env, err := s.Envs.Get(ctx, envID)
	if err != nil {
		return domain.Deployment{}, err
	}
	b, err := s.Builds.GetBuild(ctx, buildID)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && b.AppID != env.AppID) {
		return domain.Deployment{}, domain.Invalid("buildId", "build %d does not belong to app %s", buildID, env.AppName)
	}
	if err != nil {
		return domain.Deployment{}, err
	}
	if b.Status != domain.BuildSucceeded {
		return domain.Deployment{}, domain.Invalid("buildId", "build %d is %s; only succeeded builds can be deployed", buildID, b.Status)
	}
	sv, err := s.Specs.Latest(ctx, env.AppID)
	if err != nil {
		return domain.Deployment{}, err
	}
	secretVersion := 0
	if v, err := s.Secrets.Latest(ctx, secrets.EnvScope(env.ID)); err == nil {
		secretVersion = v.Version
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Deployment{}, err
	}

	d, err := s.Store.CreateDeployment(ctx, domain.Deployment{
		EnvID: env.ID, BuildID: b.ID, SpecVersion: sv.Version, SecretVersion: secretVersion,
		TriggeredBy: actor, CreatedAt: s.Now().UTC(),
	})
	if errors.Is(err, domain.ErrConflict) {
		return domain.Deployment{}, domain.Conflictf("a deployment is already in progress in %s/%s", env.AppName, env.Name)
	}
	if err != nil {
		return domain.Deployment{}, err
	}
	s.Runner.Submit(d.ID)
	return d, nil
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
