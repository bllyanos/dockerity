package imagegc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
)

// DefaultKeep is how many recent deployments per environment, and recent
// builds per app, keep their images.
const DefaultKeep = 3

// deploymentScan bounds how far back an environment's history is searched
// for succeeded deployments.
const deploymentScan = 100

// Store reads what the images belong to.
type Store interface {
	ListApps(ctx context.Context) ([]domain.App, error)
	ListAllEnvironments(ctx context.Context) ([]domain.Environment, error)
	ListDeployments(ctx context.Context, envID int64, limit int) ([]domain.Deployment, error)
	ListBuilds(ctx context.Context, appID int64, limit int) ([]domain.Build, error)
	ListBuildImages(ctx context.Context, buildID int64) ([]domain.BuildImage, error)
}

// Docker lists and removes images and lists containers.
type Docker interface {
	ListImages(ctx context.Context, labels map[string]string) ([]dockerx.ImageInfo, error)
	RemoveImage(ctx context.Context, ref string) error
	ListManaged(ctx context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error)
}

// Collector removes images of builds no recent deployment or build needs.
// The images of the last Keep builds of an app are kept as well, so a build
// that has not been deployed yet (or is still running) survives.
type Collector struct {
	Store  Store
	Docker Docker
	Keep   int
	Logger *slog.Logger
}

// Collect runs a collection over every app.
func (c *Collector) Collect(ctx context.Context) error {
	apps, err := c.Store.ListApps(ctx)
	if err != nil {
		return err
	}
	envs, err := c.Store.ListAllEnvironments(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, app := range apps {
		if err := c.collectApp(ctx, app, envs); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c *Collector) collectApp(ctx context.Context, app domain.App, envs []domain.Environment) error {
	keep, err := c.keep(ctx, app, envs)
	if err != nil {
		return err
	}
	byApp := map[string]string{domain.LabelApp: app.Name}
	containers, err := c.Docker.ListManaged(ctx, byApp)
	if err != nil {
		return err
	}
	for _, ct := range containers {
		keep.InUse[ct.Image] = true
		keep.InUse[ct.ImageID] = true
	}
	images, err := c.Docker.ListImages(ctx, byApp)
	if err != nil {
		return err
	}
	list := make([]Image, 0, len(images))
	for _, img := range images {
		list = append(list, Image{ID: img.ID, Tags: img.Tags})
	}
	var errs []error
	for _, tag := range Select(list, keep) {
		if err := c.Docker.RemoveImage(ctx, tag); err != nil {
			errs = append(errs, err)
			continue
		}
		c.Logger.Info("removed image", "app", app.Name, "image", tag)
	}
	return errors.Join(errs...)
}

// keep collects the tags of the app's last builds, of each environment's last
// succeeded deployments and of deployments in progress.
func (c *Collector) keep(ctx context.Context, app domain.App, envs []domain.Environment) (Keep, error) {
	n := c.Keep
	if n <= 0 {
		n = DefaultKeep
	}
	builds := map[int64]bool{}
	recent, err := c.Store.ListBuilds(ctx, app.ID, n)
	if err != nil {
		return Keep{}, err
	}
	for _, b := range recent {
		builds[b.ID] = true
	}
	for _, env := range envs {
		if env.AppID != app.ID {
			continue
		}
		deps, err := c.Store.ListDeployments(ctx, env.ID, deploymentScan)
		if err != nil {
			return Keep{}, err
		}
		kept := 0
		for _, d := range deps {
			switch {
			case d.Status == domain.DeploymentPending || d.Status == domain.DeploymentRunning:
				builds[d.BuildID] = true // e.g. a rollback to an old build
			case d.Status == domain.DeploymentSucceeded && kept < n:
				builds[d.BuildID] = true
				kept++
			}
		}
	}
	out := Keep{Tags: map[string]bool{}, InUse: map[string]bool{}}
	for id := range builds {
		imgs, err := c.Store.ListBuildImages(ctx, id)
		if err != nil {
			return Keep{}, err
		}
		for _, img := range imgs {
			out.Tags[img.ImageTag] = true
		}
	}
	return out, nil
}
