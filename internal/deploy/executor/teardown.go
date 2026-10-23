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
	ListVolumes(ctx context.Context, labels map[string]string) ([]string, error)
	RemoveVolume(ctx context.Context, name string) error
}

// Teardown removes an environment's containers and network. Volumes are kept:
// they hold data and are backed up and removed by the environment delete flow.
type Teardown struct {
	Docker TeardownDocker
	// Routes, when set, has the environment's route file removed first.
	Routes RouteFiles
}

// Remove stops and removes everything Dockerity runs for app/env.
func (t Teardown) Remove(ctx context.Context, app, env string) error {
	if t.Routes != nil {
		if err := t.Routes.Remove(app, env); err != nil {
			return err
		}
	}
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

// RemoveVolumes deletes the environment's volumes. Back them up first.
func (t Teardown) RemoveVolumes(ctx context.Context, app, env string) error {
	names, err := t.Docker.ListVolumes(ctx, map[string]string{domain.LabelApp: app, domain.LabelEnv: env})
	if err != nil {
		return err
	}
	for _, n := range names {
		if err := t.Docker.RemoveVolume(ctx, n); err != nil {
			return err
		}
	}
	return nil
}
