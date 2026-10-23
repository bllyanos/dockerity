package environments

import (
	"context"
	"sort"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
)

// Container is one managed container of an environment, as shown in the UI.
type Container struct {
	Name       string
	Service    string
	Deployment string
	Image      string
	Running    bool
	Routed     bool
}

// Containers lists what runs in an environment, including whether Traefik
// sends it traffic, so both versions are visible during a blue-green switch.
type Containers struct {
	Envs interface {
		Get(ctx context.Context, id int64) (domain.Environment, error)
	}
	Docker interface {
		ListManaged(ctx context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error)
	}
	Routes interface {
		Routed(app, env string) (map[string]bool, error)
	}
}

// List returns the environment's containers ordered by service and name.
func (c *Containers) List(ctx context.Context, envID int64) ([]Container, error) {
	env, err := c.Envs.Get(ctx, envID)
	if err != nil {
		return nil, err
	}
	list, err := c.Docker.ListManaged(ctx, map[string]string{domain.LabelApp: env.AppName, domain.LabelEnv: env.Name})
	if err != nil {
		return nil, err
	}
	routed, err := c.Routes.Routed(env.AppName, env.Name)
	if err != nil {
		return nil, err
	}
	out := make([]Container, 0, len(list))
	for _, ci := range list {
		out = append(out, Container{
			Name: ci.Name, Service: ci.Labels[domain.LabelService], Deployment: ci.Labels[domain.LabelDeployment],
			Image: ci.Image, Running: ci.Running, Routed: routed[ci.Name],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
