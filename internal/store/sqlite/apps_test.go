package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestAppsCRUD(t *testing.T) {
	r := Apps{DB: migratedDB(t)}
	ctx := context.Background()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newApp := func(name string) domain.App {
		return domain.App{Name: name, RepoURL: "git@github.com:org/" + name + ".git", DefaultBranch: "main", CreatedAt: t0, UpdatedAt: t0}
	}

	shop, err := r.CreateApp(ctx, newApp("shop"))
	if err != nil || shop.ID == 0 {
		t.Fatalf("create: %+v %v", shop, err)
	}
	if _, err := r.CreateApp(ctx, newApp("shop")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	_, _ = r.CreateApp(ctx, newApp("blog"))

	list, err := r.ListApps(ctx)
	if err != nil || len(list) != 2 || list[0].Name != "blog" {
		t.Fatalf("list: %+v %v", list, err)
	}

	shop.DefaultBranch, shop.DeployPublicKey, shop.UpdatedAt = "develop", "ssh-ed25519 AAAA", t0.Add(time.Hour)
	if err := r.UpdateApp(ctx, shop); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetApp(ctx, shop.ID)
	if err != nil || got != shop {
		t.Fatalf("get after update: %+v %v", got, err)
	}

	if err := r.DeleteApp(ctx, shop.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetApp(ctx, shop.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	if err := r.DeleteApp(ctx, shop.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}
