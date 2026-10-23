package warning

import (
	"slices"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/planner"
	"github.com/bllyanos/dockerity/internal/spec"
)

func TestTagAndMajor(t *testing.T) {
	cases := []struct {
		image, tag string
		major      int
		ok         bool
	}{
		{"mysql:8.4", "8.4", 8, true},
		{"mysql", "latest", 0, false},
		{"registry.local:5000/team/app:v2.1.0", "v2.1.0", 2, true},
		{"registry.local:5000/team/app", "latest", 0, false},
		{"postgres:16-alpine@sha256:abc", "16-alpine", 16, true},
		{"dkt/shop/api:abc123", "abc123", 0, false},
	}
	for _, c := range cases {
		tag := Tag(c.image)
		major, ok := Major(tag)
		if tag != c.tag || major != c.major || ok != c.ok {
			t.Errorf("%s: tag %q major %d %v", c.image, tag, major, ok)
		}
	}
	if !MajorChange("mysql:8.4", "mysql:9.1") || MajorChange("mysql:8.0", "mysql:8.4") || MajorChange("app:abc", "app:9") {
		t.Fatal("MajorChange")
	}
}

func TestBuild(t *testing.T) {
	s, err := spec.Load([]byte(`
services:
  db:
    kind: stateful
    image: mysql:9.1
    volumes:
      data: /var/lib/mysql
  api:
    kind: deployment
    image: api:2
    depends_on: [db]
    http: {port: 80}
    healthcheck: {http: /h}
`))
	if err != nil {
		t.Fatal(err)
	}
	current := map[string]planner.Running{"db": {Image: "mysql:8.4"}, "api": {Image: "api:1"}}
	plan, err := planner.Make(planner.Input{App: "shop", Env: "prod", Spec: s, Current: current})
	if err != nil {
		t.Fatal(err)
	}
	last := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	w := Build(Input{App: "shop", Env: "prod", Spec: s, Plan: plan, Current: current,
		LastBackups: map[string]time.Time{"dkt-shop-prod-db-data": last}})

	if !w.Downtime || len(w.Services) != 2 {
		t.Fatalf("warning %+v", w)
	}
	db, api := w.Services[0], w.Services[1]
	if db.Name != "db" || !db.Downtime || !db.MajorChange || !db.Backup || db.OldImage != "mysql:8.4" ||
		!slices.Equal(db.Dependents, []string{"api"}) || !db.LastBackup["dkt-shop-prod-db-data"].Equal(last) {
		t.Fatalf("db %+v", db)
	}
	if api.Downtime || api.Strategy != spec.StrategyBlueGreen || api.Backup || len(api.Dependents) != 0 {
		t.Fatalf("api %+v", api)
	}
}
