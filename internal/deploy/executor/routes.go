package executor

import (
	"context"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/routing"
	"github.com/bllyanos/dockerity/internal/spec"
)

// RouteFiles stores Traefik file-provider configuration per environment.
type RouteFiles interface {
	Write(app, env string, data []byte) error
	Remove(app, env string) error
}

// AllEnvs lists every environment, to rebuild routes on startup.
type AllEnvs interface {
	ListAllEnvironments(ctx context.Context) ([]domain.Environment, error)
}

// syncRoutes writes the environment's routes from what runs in Docker: every
// running container of an HTTP service in s, except excluded ones (not yet
// healthy, or draining before a stop). Without RouteFiles it does nothing.
func (e *Executor) syncRoutes(ctx context.Context, env domain.Environment, s spec.Spec, excluded map[string]bool) error {
	if e.Routes == nil {
		return nil
	}
	running, err := e.Docker.ListManaged(ctx, map[string]string{domain.LabelApp: env.AppName, domain.LabelEnv: env.Name})
	if err != nil {
		return err
	}
	byService := map[string][]string{}
	for _, c := range running {
		if c.Running && !excluded[c.Name] {
			svc := c.Labels[domain.LabelService]
			byService[svc] = append(byService[svc], c.Name)
		}
	}
	routes := routing.Env{App: env.AppName, Env: env.Name, Domain: env.Domain}
	for _, svc := range s.Services {
		if svc.HTTP == nil || svc.Kind == spec.KindJob {
			continue
		}
		routes.Services = append(routes.Services, routing.Service{
			Name: svc.Name, Path: svc.HTTP.RoutePath(), Port: svc.HTTP.Port, Containers: byService[svc.Name],
		})
	}
	data, err := routing.Config(routes, e.Routing)
	if err != nil {
		return err
	}
	return e.Routes.Write(env.AppName, env.Name, data)
}

// RebuildRoutes rewrites an environment's routes from its current deployment
// and what runs in Docker, e.g. after a restart or a domain change.
func (e *Executor) RebuildRoutes(ctx context.Context, envID int64) error {
	env, err := e.Envs.Get(ctx, envID)
	if err != nil {
		return err
	}
	var s spec.Spec
	cur, err := e.Store.CurrentDeployment(ctx, envID)
	switch {
	case err == nil:
		sv, err := e.Specs.Get(ctx, env.AppID, cur.SpecVersion)
		if err != nil {
			return err
		}
		if s, err = spec.Load([]byte(sv.YAML)); err != nil {
			return err
		}
	case !errors.Is(err, domain.ErrNotFound):
		return err
	}
	return e.syncRoutes(ctx, env, s, nil)
}

// RebuildAllRoutes rewrites the routes of every environment.
func (e *Executor) RebuildAllRoutes(ctx context.Context) error {
	if e.AllEnvs == nil || e.Routes == nil {
		return nil
	}
	envs, err := e.AllEnvs.ListAllEnvironments(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, env := range envs {
		if err := e.RebuildRoutes(ctx, env.ID); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
