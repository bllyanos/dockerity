package deployments

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

type memStore struct{ deps []domain.Deployment }

func (m *memStore) CreateDeployment(_ context.Context, d domain.Deployment) (domain.Deployment, error) {
	for _, x := range m.deps {
		if x.EnvID == d.EnvID && !x.Status.Terminal() {
			return domain.Deployment{}, domain.ErrConflict
		}
	}
	d.ID, d.Status = int64(len(m.deps)+1), domain.DeploymentPending
	m.deps = append(m.deps, d)
	return d, nil
}

func (m *memStore) GetDeployment(_ context.Context, id int64) (domain.Deployment, error) {
	if id < 1 || int(id) > len(m.deps) {
		return domain.Deployment{}, domain.ErrNotFound
	}
	return m.deps[id-1], nil
}

func (m *memStore) ListDeployments(context.Context, int64, int) ([]domain.Deployment, error) {
	return m.deps, nil
}

func (m *memStore) ListSteps(context.Context, int64) ([]domain.DeploymentStep, error) {
	return nil, nil
}

type envs struct{}

func (envs) Get(_ context.Context, id int64) (domain.Environment, error) {
	if id != 1 {
		return domain.Environment{}, domain.ErrNotFound
	}
	return domain.Environment{ID: 1, AppID: 1, AppName: "shop", Name: "staging"}, nil
}

type builds map[int64]domain.Build

func (b builds) GetBuild(_ context.Context, id int64) (domain.Build, error) {
	if x, ok := b[id]; ok {
		return x, nil
	}
	return domain.Build{}, domain.ErrNotFound
}

type specs struct{}

func (specs) Latest(context.Context, int64) (domain.AppSpecVersion, error) {
	return domain.AppSpecVersion{Version: 4}, nil
}

type secretVersions struct{ version int }

func (s secretVersions) Latest(context.Context, secrets.Scope) (secrets.Version, error) {
	if s.version == 0 {
		return secrets.Version{}, domain.ErrNotFound
	}
	return secrets.Version{Version: s.version}, nil
}

type runner struct{ submitted []int64 }

func (r *runner) Submit(id int64) { r.submitted = append(r.submitted, id) }

func newService(secretVersion int) (*Service, *runner) {
	r := &runner{}
	return &Service{
		Store: &memStore{}, Envs: envs{}, Specs: specs{}, Secrets: secretVersions{secretVersion}, Runner: r, Now: time.Now,
		Builds: builds{
			1: {ID: 1, AppID: 1, Status: domain.BuildSucceeded},
			2: {ID: 2, AppID: 1, Status: domain.BuildFailed},
			3: {ID: 3, AppID: 2, Status: domain.BuildSucceeded},
		},
	}, r
}

func TestStart(t *testing.T) {
	s, r := newService(7)
	ctx := context.Background()
	d, err := s.Start(ctx, 1, 1, nil)
	if err != nil || d.SpecVersion != 4 || d.SecretVersion != 7 || d.BuildID != 1 || len(r.submitted) != 1 {
		t.Fatalf("start: %+v %v %v", d, err, r.submitted)
	}
	if _, err := s.Start(ctx, 1, 1, nil); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("concurrent deploy: %v", err)
	}
}

func TestStartRejects(t *testing.T) {
	s, r := newService(0)
	ctx := context.Background()
	var ve *domain.ValidationError
	for name, buildID := range map[string]int64{"failed build": 2, "other app": 3, "missing": 9} {
		if _, err := s.Start(ctx, 1, buildID, nil); !errors.As(err, &ve) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := s.Start(ctx, 9, 1, nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing env: %v", err)
	}
	d, err := s.Start(ctx, 1, 1, nil)
	if err != nil || d.SecretVersion != 0 {
		t.Fatalf("no secrets yet: %+v %v", d, err)
	}
	if len(r.submitted) != 1 {
		t.Fatal("rejected deployments must not be submitted")
	}
}
