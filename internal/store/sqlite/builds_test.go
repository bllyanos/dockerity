package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func newAppForTest(t *testing.T, r Apps, name string) domain.App {
	t.Helper()
	a, err := r.CreateApp(context.Background(), domain.App{Name: name, RepoURL: "git@h:o/r.git", DefaultBranch: "main", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestBuildLifecycle(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	app := newAppForTest(t, Apps{DB: db}, "shop")
	r := Builds{DB: db}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	b, err := r.CreateBuild(ctx, domain.Build{AppID: app.ID, Ref: "main", CreatedAt: t0})
	if err != nil || b.Status != domain.BuildQueued {
		t.Fatalf("create: %+v %v", b, err)
	}
	if err := r.FinishBuild(ctx, b.ID, domain.BuildSucceeded, "", t0); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("queued -> succeeded must conflict: %v", err)
	}
	if err := r.StartBuild(ctx, b.ID, "abc123", t0.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := r.StartBuild(ctx, b.ID, "abc123", t0); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("double start must conflict: %v", err)
	}
	if err := r.AddBuildImage(ctx, domain.BuildImage{BuildID: b.ID, Service: "api", ImageTag: "dkt/shop/api:abc123", ImageID: "sha256:1"}); err != nil {
		t.Fatal(err)
	}
	if err := r.FinishBuild(ctx, b.ID, domain.BuildSucceeded, "", t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := r.FinishBuild(ctx, b.ID, domain.BuildFailed, "x", t0); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("finished build must not change: %v", err)
	}

	got, err := r.GetBuild(ctx, b.ID)
	if err != nil || got.Status != domain.BuildSucceeded || got.CommitSHA != "abc123" || got.StartedAt == nil || got.FinishedAt == nil {
		t.Fatalf("get: %+v %v", got, err)
	}
	imgs, _ := r.ListBuildImages(ctx, b.ID)
	if len(imgs) != 1 || imgs[0].ImageTag != "dkt/shop/api:abc123" {
		t.Fatalf("images: %+v", imgs)
	}
	if err := r.FinishBuild(ctx, 999, domain.BuildCanceled, "", t0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing build: %v", err)
	}
	if err := r.FinishBuild(ctx, b.ID, domain.BuildRunning, "", t0); err == nil {
		t.Fatal("non-terminal finish accepted")
	}
}

func TestListBuilds(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	app := newAppForTest(t, Apps{DB: db}, "shop")
	r := Builds{DB: db}
	for range 3 {
		_, _ = r.CreateBuild(ctx, domain.Build{AppID: app.ID, Ref: "main", CreatedAt: time.Now()})
	}
	_ = r.StartBuild(ctx, 2, "sha", time.Now())

	list, _ := r.ListBuilds(ctx, app.ID, 2)
	if len(list) != 2 || list[0].ID != 3 {
		t.Fatalf("list: %+v", list)
	}
	queued, _ := r.ListBuildsByStatus(ctx, domain.BuildQueued)
	running, _ := r.ListBuildsByStatus(ctx, domain.BuildRunning)
	if len(queued) != 2 || queued[0].ID != 1 || len(running) != 1 || running[0].ID != 2 {
		t.Fatalf("by status: %+v %+v", queued, running)
	}
}
