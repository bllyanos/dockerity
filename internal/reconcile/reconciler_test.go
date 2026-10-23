package reconcile

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/bllyanos/dockerity/internal/deploy/drift"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
)

const specYAML = `
services:
  web:
    kind: deployment
    image: nginx:1
    strategy: recreate
    http: {port: 80}
  db:
    kind: stateful
    image: mysql:8
    volumes: {data: /var/lib/mysql}
  migrate:
    kind: job
    image: nginx:1
`

type envs struct{ list []domain.Environment }

func (e envs) Get(_ context.Context, id int64) (domain.Environment, error) {
	for _, env := range e.list {
		if env.ID == id {
			return env, nil
		}
	}
	return domain.Environment{}, domain.ErrNotFound
}

func (e envs) ListAllEnvironments(context.Context) ([]domain.Environment, error) { return e.list, nil }

type deployments struct {
	current map[int64]int64
	active  []domain.Deployment
}

func (d deployments) CurrentDeployment(_ context.Context, envID int64) (domain.Deployment, error) {
	id, ok := d.current[envID]
	if !ok {
		return domain.Deployment{}, domain.ErrNotFound
	}
	return domain.Deployment{ID: id, EnvID: envID, SpecVersion: 1}, nil
}

func (d deployments) ListDeploymentsByStatus(_ context.Context, st domain.DeploymentStatus) ([]domain.Deployment, error) {
	if st == domain.DeploymentRunning {
		return d.active, nil
	}
	return nil, nil
}

type specs struct{}

func (specs) Get(context.Context, int64, int) (domain.AppSpecVersion, error) {
	return domain.AppSpecVersion{Version: 1, YAML: specYAML}, nil
}

type docker struct {
	byEnv   map[string][]dockerx.ContainerInfo
	removed []string
}

func (d *docker) ListManaged(_ context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error) {
	return d.byEnv[labels[domain.LabelEnv]], nil
}

func (d *docker) RemoveContainer(_ context.Context, id string) error {
	d.removed = append(d.removed, id)
	return nil
}

type routes struct {
	routed  map[string]map[string]bool
	rebuilt []int64
}

func (r *routes) Routed(_, env string) (map[string]bool, error) { return r.routed[env], nil }

func (r *routes) RebuildRoutes(_ context.Context, envID int64) error {
	r.rebuilt = append(r.rebuilt, envID)
	return nil
}

func container(id, svc, dep string, running bool, extra ...string) dockerx.ContainerInfo {
	labels := map[string]string{domain.LabelService: svc, domain.LabelDeployment: dep}
	for i := 0; i+1 < len(extra); i += 2 {
		labels[extra[i]] = extra[i+1]
	}
	return dockerx.ContainerInfo{ID: id, Name: "c-" + id, Labels: labels, Running: running}
}

func setup() (*Reconciler, *docker, *routes) {
	d := &docker{byEnv: map[string][]dockerx.ContainerInfo{
		"prod": {
			container("web5", "web", "5", true),
			container("db3", "db", "3", true),
			container("web4", "web", "4", false),                            // stopped leftover
			container("sys", "web", "1", false, domain.LabelSystem, "true"), // never touched
		},
		"staging": {
			container("s-web2", "web", "2", false), // expected but stopped
			container("s-old", "web", "1", false),
		},
	}}
	rt := &routes{routed: map[string]map[string]bool{"prod": {"c-web5": true, "c-web4": true}}}
	r := &Reconciler{
		Envs: envs{list: []domain.Environment{
			{ID: 1, AppID: 1, AppName: "shop", Name: "prod"},
			{ID: 2, AppID: 1, AppName: "shop", Name: "staging"},
		}},
		Deployments: deployments{current: map[int64]int64{1: 5, 2: 2}},
		Specs:       specs{}, Docker: d, Routes: rt, Rebuilder: rt,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return r, d, rt
}

func TestCheck(t *testing.T) {
	r, d, _ := setup()
	rep, err := r.Check(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Items) != 1 || rep.Items[0].Kind != drift.KindDanglingRoute || rep.Items[0].Container != "c-web4" {
		t.Fatalf("items %+v", rep.Items)
	}
	if !slices.Equal(rep.Remove, []string{"web4"}) || len(d.removed) != 0 {
		t.Fatalf("check must only report: remove %v removed %v", rep.Remove, d.removed)
	}
}

func TestRunOnceRepairs(t *testing.T) {
	r, d, rt := setup()
	r.RunOnce(context.Background())
	slices.Sort(d.removed)
	if !slices.Equal(d.removed, []string{"s-old", "web4"}) {
		t.Fatalf("removed %v; running, expected and system containers must stay", d.removed)
	}
	if !slices.Equal(rt.rebuilt, []int64{1}) {
		t.Fatalf("rebuilt %v", rt.rebuilt)
	}
}

func TestRunOnceSkipsBusyEnvironments(t *testing.T) {
	r, d, rt := setup()
	r.Deployments = deployments{current: map[int64]int64{1: 5, 2: 2}, active: []domain.Deployment{{ID: 6, EnvID: 1}}}
	r.RunOnce(context.Background())
	if !slices.Equal(d.removed, []string{"s-old"}) || len(rt.rebuilt) != 0 {
		t.Fatalf("busy environment touched: removed %v rebuilt %v", d.removed, rt.rebuilt)
	}
}
