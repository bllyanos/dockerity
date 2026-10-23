package environments

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type memStore struct{ envs []domain.Environment }

func (m *memStore) ListEnvironments(_ context.Context, appID int64) ([]domain.Environment, error) {
	var out []domain.Environment
	for _, e := range m.envs {
		if e.AppID == appID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) GetEnvironment(_ context.Context, id int64) (domain.Environment, error) {
	for _, e := range m.envs {
		if e.ID == id {
			return e, nil
		}
	}
	return domain.Environment{}, domain.ErrNotFound
}

func (m *memStore) CreateEnvironment(_ context.Context, e domain.Environment) (domain.Environment, error) {
	for _, x := range m.envs {
		if (x.AppID == e.AppID && x.Name == e.Name) || (e.Domain != "" && x.Domain == e.Domain) {
			return domain.Environment{}, domain.ErrConflict
		}
	}
	e.ID = int64(len(m.envs) + 1)
	m.envs = append(m.envs, e)
	return e, nil
}

func (m *memStore) UpdateEnvironment(_ context.Context, e domain.Environment) error {
	m.envs[e.ID-1] = e
	return nil
}

func (m *memStore) DeleteEnvironment(_ context.Context, id int64) error {
	m.envs[id-1].ID = -1
	return nil
}

type oneApp struct{}

func (oneApp) Get(_ context.Context, id int64) (domain.App, error) {
	if id != 1 {
		return domain.App{}, domain.ErrNotFound
	}
	return domain.App{ID: 1, Name: "shop"}, nil
}

func TestEnvironments(t *testing.T) {
	s := &Service{Store: &memStore{}, Apps: oneApp{}, Now: time.Now}
	ctx := context.Background()

	e, err := s.CreatePermanent(ctx, 1, " staging ", " Staging.Shop.Test. ")
	if err != nil || e.Name != "staging" || e.Domain != "staging.shop.test" || e.Type != domain.EnvPermanent {
		t.Fatalf("create: %+v %v", e, err)
	}
	if _, err := s.CreatePermanent(ctx, 1, "staging", ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	var ve *domain.ValidationError
	if _, err := s.CreatePermanent(ctx, 1, "Prod", ""); !errors.As(err, &ve) {
		t.Fatalf("bad name: %v", err)
	}
	if _, err := s.CreatePermanent(ctx, 1, "prod", "not a domain"); !errors.As(err, &ve) {
		t.Fatalf("bad domain: %v", err)
	}
	if _, err := s.CreatePermanent(ctx, 2, "prod", ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing app: %v", err)
	}

	got, err := s.UpdateDomain(ctx, e.ID, "")
	if err != nil || got.Domain != "" {
		t.Fatalf("unroute: %+v %v", got, err)
	}
	if _, err := s.Delete(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
}

type deploymentsFake struct{ status domain.DeploymentStatus }

func (d deploymentsFake) ListDeployments(context.Context, int64, int) ([]domain.Deployment, error) {
	if d.status == "" {
		return nil, nil
	}
	return []domain.Deployment{{Status: d.status}}, nil
}

type runtimeFake struct{ removed []string }

func (r *runtimeFake) Remove(_ context.Context, app, env string) error {
	r.removed = append(r.removed, app+"/"+env)
	return nil
}

func TestDeleteGuardsAndTearsDown(t *testing.T) {
	rt := &runtimeFake{}
	s := &Service{Store: &memStore{}, Apps: oneApp{}, Now: time.Now, Runtime: rt, Deployments: deploymentsFake{status: domain.DeploymentRunning}}
	ctx := context.Background()
	e, _ := s.CreatePermanent(ctx, 1, "staging", "")
	e.AppName = "shop"
	s.Store.(*memStore).envs[0].AppName = "shop"

	if _, err := s.Delete(ctx, e.ID); !errors.Is(err, domain.ErrConflict) || len(rt.removed) != 0 {
		t.Fatalf("delete during deployment: %v %v", err, rt.removed)
	}
	s.Deployments = deploymentsFake{status: domain.DeploymentSucceeded}
	if _, err := s.Delete(ctx, e.ID); err != nil || len(rt.removed) != 1 || rt.removed[0] != "shop/staging" {
		t.Fatalf("delete: %v %v", err, rt.removed)
	}
}
