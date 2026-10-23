package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestSpecVersions(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	app, _ := Apps{DB: db}.CreateApp(ctx, domain.App{Name: "shop", RepoURL: "git@h:o/r.git", DefaultBranch: "main", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	r := SpecVersions{DB: db}

	if _, err := r.LatestSpecVersion(ctx, app.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty: %v", err)
	}
	for v := 1; v <= 2; v++ {
		if err := r.InsertSpecVersion(ctx, domain.AppSpecVersion{AppID: app.ID, Version: v, YAML: "v" + string(rune('0'+v)), CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.InsertSpecVersion(ctx, domain.AppSpecVersion{AppID: app.ID, Version: 2, YAML: "dup", CreatedAt: time.Now()}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	latest, _ := r.LatestSpecVersion(ctx, app.ID)
	first, _ := r.GetSpecVersion(ctx, app.ID, 1)
	if latest.Version != 2 || latest.YAML != "v2" || first.YAML != "v1" {
		t.Fatalf("latest %+v first %+v", latest, first)
	}
	list, err := r.ListSpecVersions(ctx, app.ID)
	if err != nil || len(list) != 2 || list[0].Version != 2 || list[0].YAML != "" {
		t.Fatalf("list %+v %v", list, err)
	}

	if err := (Apps{DB: db}).DeleteApp(ctx, app.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := r.ListSpecVersions(ctx, app.ID); len(list) != 0 {
		t.Fatal("versions must be removed with the app")
	}
}
