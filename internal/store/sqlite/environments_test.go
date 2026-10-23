package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestEnvironmentsCRUD(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	shop := newAppForTest(t, Apps{DB: db}, "shop")
	blog := newAppForTest(t, Apps{DB: db}, "blog")
	r := Environments{DB: db}
	env := func(appID int64, name, dom string) domain.Environment {
		return domain.Environment{AppID: appID, Name: name, Type: domain.EnvPermanent, Domain: dom, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	}

	staging, err := r.CreateEnvironment(ctx, env(shop.ID, "staging", "staging.shop.test"))
	if err != nil || staging.AppName != "shop" || staging.Type != domain.EnvPermanent {
		t.Fatalf("create: %+v %v", staging, err)
	}
	if _, err := r.CreateEnvironment(ctx, env(shop.ID, "staging", "")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := r.CreateEnvironment(ctx, env(blog.ID, "prod", "staging.shop.test")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate domain: %v", err)
	}
	if _, err := r.CreateEnvironment(ctx, env(blog.ID, "staging", "")); err != nil {
		t.Fatalf("same name in another app: %v", err)
	}
	if _, err := r.CreateEnvironment(ctx, env(blog.ID, "internal", "")); err != nil {
		t.Fatalf("two unrouted environments: %v", err)
	}

	staging.Domain = "stg.shop.test"
	if err := r.UpdateEnvironment(ctx, staging); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.GetEnvironment(ctx, staging.ID); got.Domain != "stg.shop.test" {
		t.Fatalf("update: %+v", got)
	}
	list, _ := r.ListEnvironments(ctx, blog.ID)
	if len(list) != 2 || list[0].Name != "internal" {
		t.Fatalf("list: %+v", list)
	}
	if err := r.DeleteEnvironment(ctx, staging.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetEnvironment(ctx, staging.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestLimitsColumns(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	apps := Apps{DB: db}
	app := newAppForTest(t, apps, "shop")
	app.MaxEphemeral = 2
	if err := apps.UpdateApp(ctx, app); err != nil {
		t.Fatal(err)
	}
	if got, _ := apps.GetApp(ctx, app.ID); got.MaxEphemeral != 2 {
		t.Fatalf("max ephemeral %d", got.MaxEphemeral)
	}

	r := Environments{DB: db}
	now := time.Now()
	e, _ := r.CreateEnvironment(ctx, domain.Environment{AppID: app.ID, Name: "pr-1", Type: domain.EnvEphemeral, ExpiresAt: &now, CreatedAt: now, UpdatedAt: now})
	_, _ = r.CreateEnvironment(ctx, domain.Environment{AppID: app.ID, Name: "prod", Type: domain.EnvPermanent, CreatedAt: now, UpdatedAt: now})
	if n, err := r.CountEphemeral(ctx, app.ID); err != nil || n != 1 {
		t.Fatalf("count %d %v", n, err)
	}

	o := domain.ResourceOverrides{"*": {Memory: "256m"}, "db": {CPUs: "2"}}
	if err := r.SetResources(ctx, e.ID, o); err != nil {
		t.Fatal(err)
	}
	got, _ := r.GetEnvironment(ctx, e.ID)
	if got.Resources["db"].CPUs != "2" || got.Resources["*"].Memory != "256m" {
		t.Fatalf("overrides %+v", got.Resources)
	}
}
