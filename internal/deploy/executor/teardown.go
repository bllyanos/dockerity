package executor

import (
	"context"
	"time"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
)

// TeardownDocker is what removing an environment's runtime needs.
type TeardownDocker interface {
	ListManaged(ctx context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error)
	StopContainer(ctx context.Context, id string, grace time.Duration) error
	RemoveContainer(ctx context.Context, id string) error
	RemoveNetwork(ctx context.Context, name string) error
}

// Teardown removes an environment's containers and network. Volumes are kept:
// they hold data and are backed up and removed by the environment delete flow.
type Teardown struct {
	Docker TeardownDocker
}

// Remove stops and removes everything Dockerity runs for app/env.
func (t Teardown) Remove(ctx context.Context, app, env string) error {
	list, err := t.Docker.ListManaged(ctx, map[string]string{domain.LabelApp: app, domain.LabelEnv: env})
	if err != nil {
		return err
	}
	for _, c := range list {
		if err := t.Docker.StopContainer(ctx, c.ID, 30*time.Second); err != nil {
			return err
		}
		if err := t.Docker.RemoveContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	return t.Docker.RemoveNetwork(ctx, domain.NetworkName(app, env))
}
