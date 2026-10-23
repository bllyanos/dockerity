package environments

import (
	"context"
	"testing"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
)

type fixedEnv struct{}

func (fixedEnv) Get(_ context.Context, id int64) (domain.Environment, error) {
	return domain.Environment{ID: id, AppName: "shop", Name: "prod"}, nil
}

type listDocker struct{ got map[string]string }

func (l *listDocker) ListManaged(_ context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error) {
	l.got = labels
	return []dockerx.ContainerInfo{
		{Name: "dkt-shop-prod-web-d2", Image: "web:b", Running: true, Labels: map[string]string{"dockerity.service": "web", "dockerity.deployment": "2"}},
		{Name: "dkt-shop-prod-db-d1", Image: "mysql", Running: true, Labels: map[string]string{"dockerity.service": "db", "dockerity.deployment": "1"}},
		{Name: "dkt-shop-prod-web-d1", Image: "web:a", Running: true, Labels: map[string]string{"dockerity.service": "web", "dockerity.deployment": "1"}},
	}, nil
}

type routedSet map[string]bool

func (r routedSet) Routed(string, string) (map[string]bool, error) { return r, nil }

func TestContainers(t *testing.T) {
	d := &listDocker{}
	c := &Containers{Envs: fixedEnv{}, Docker: d, Routes: routedSet{"dkt-shop-prod-web-d1": true, "dkt-shop-prod-web-d2": true}}
	list, err := c.List(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.got["dockerity.app"] != "shop" || d.got["dockerity.env"] != "prod" {
		t.Fatalf("filter %v", d.got)
	}
	if len(list) != 3 || list[0].Service != "db" || list[0].Routed || list[1].Name != "dkt-shop-prod-web-d1" || !list[1].Routed || !list[2].Routed {
		t.Fatalf("list %+v", list)
	}
}
