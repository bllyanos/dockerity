package imagegc

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
)

type store struct{}

func (store) ListApps(context.Context) ([]domain.App, error) {
	return []domain.App{{ID: 1, Name: "shop"}}, nil
}

func (store) ListAllEnvironments(context.Context) ([]domain.Environment, error) {
	return []domain.Environment{{ID: 10, AppID: 1, Name: "prod"}, {ID: 20, AppID: 2, Name: "other"}}, nil
}

// prod history, newest first: builds 6 (running), 5, 4 (failed), 3, 2, 1.
func (store) ListDeployments(_ context.Context, envID int64, _ int) ([]domain.Deployment, error) {
	if envID != 10 {
		return []domain.Deployment{{BuildID: 1, Status: domain.DeploymentSucceeded}}, nil
	}
	return []domain.Deployment{
		{BuildID: 1, Status: domain.DeploymentRunning}, // rollback in progress
		{BuildID: 5, Status: domain.DeploymentSucceeded},
		{BuildID: 4, Status: domain.DeploymentFailed},
		{BuildID: 3, Status: domain.DeploymentSucceeded},
		{BuildID: 2, Status: domain.DeploymentSucceeded},
		{BuildID: 1, Status: domain.DeploymentSucceeded},
	}, nil
}

func (store) ListBuilds(context.Context, int64, int) ([]domain.Build, error) {
	return []domain.Build{{ID: 9}, {ID: 8}}, nil
}

func (store) ListBuildImages(_ context.Context, id int64) ([]domain.BuildImage, error) {
	return []domain.BuildImage{{BuildID: id, Service: "web", ImageTag: tag(id)}}, nil
}

func tag(build int64) string { return "dkt/shop/web:b" + string(rune('0'+build)) }

type docker struct{ removed []string }

func (d *docker) ListImages(_ context.Context, labels map[string]string) ([]dockerx.ImageInfo, error) {
	if labels[domain.LabelApp] != "shop" {
		return nil, nil
	}
	var out []dockerx.ImageInfo
	for b := int64(1); b <= 9; b++ {
		out = append(out, dockerx.ImageInfo{ID: "sha256:" + tag(b), Tags: []string{tag(b)}})
	}
	out = append(out, dockerx.ImageInfo{ID: "sha256:in-use", Tags: []string{"dkt/shop/web:manual"}})
	return out, nil
}

func (d *docker) RemoveImage(_ context.Context, ref string) error {
	d.removed = append(d.removed, ref)
	return nil
}

func (d *docker) ListManaged(context.Context, map[string]string) ([]dockerx.ContainerInfo, error) {
	return []dockerx.ContainerInfo{{Image: "dkt/shop/web:b4", ImageID: "sha256:in-use"}}, nil
}

func TestCollect(t *testing.T) {
	d := &docker{}
	c := &Collector{Store: store{}, Docker: d, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := c.Collect(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Kept: running rollback (1), last three succeeded (5, 3, 2), recent builds
	// (9, 8), used by a container (4 by tag, "manual" by ID).
	want := []string{tag(6), tag(7)}
	slices.Sort(d.removed)
	if !slices.Equal(d.removed, want) {
		t.Fatalf("removed %v want %v", d.removed, want)
	}
}
