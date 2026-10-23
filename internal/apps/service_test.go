package apps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type memStore struct{ apps []domain.App }

func (m *memStore) ListApps(context.Context) ([]domain.App, error) { return m.apps, nil }

func (m *memStore) GetApp(_ context.Context, id int64) (domain.App, error) {
	for _, a := range m.apps {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.App{}, domain.ErrNotFound
}

func (m *memStore) CreateApp(_ context.Context, a domain.App) (domain.App, error) {
	for _, x := range m.apps {
		if x.Name == a.Name {
			return domain.App{}, domain.ErrConflict
		}
	}
	a.ID = int64(len(m.apps) + 1)
	m.apps = append(m.apps, a)
	return a, nil
}

func (m *memStore) UpdateApp(_ context.Context, a domain.App) error {
	for i := range m.apps {
		if m.apps[i].ID == a.ID {
			m.apps[i] = a
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *memStore) DeleteApp(_ context.Context, id int64) error {
	for i := range m.apps {
		if m.apps[i].ID == id {
			m.apps = append(m.apps[:i], m.apps[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func newService() *Service {
	return &Service{Store: &memStore{}, Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}
}

func TestCreate(t *testing.T) {
	s := newService()
	ctx := context.Background()
	a, err := s.Create(ctx, CreateInput{Name: " shop ", RepoURL: "git@github.com:org/shop.git"})
	if err != nil || a.Name != "shop" || a.DefaultBranch != "main" || a.CreatedAt.IsZero() {
		t.Fatalf("create: %+v %v", a, err)
	}
	if _, err := s.Create(ctx, CreateInput{Name: "shop", RepoURL: "git@github.com:org/shop.git"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	for name, in := range map[string]CreateInput{
		"name":   {Name: "Shop", RepoURL: "git@github.com:org/shop.git"},
		"repo":   {Name: "blog", RepoURL: "https://github.com/org/blog"},
		"branch": {Name: "blog", RepoURL: "git@github.com:org/blog.git", DefaultBranch: "a..b"},
	} {
		var ve *domain.ValidationError
		if _, err := s.Create(ctx, in); !errors.As(err, &ve) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestUpdateAndDelete(t *testing.T) {
	s := newService()
	ctx := context.Background()
	a, _ := s.Create(ctx, CreateInput{Name: "shop", RepoURL: "git@github.com:org/shop.git"})

	branch := "develop"
	got, err := s.Update(ctx, a.ID, UpdateInput{DefaultBranch: &branch})
	if err != nil || got.DefaultBranch != "develop" || got.RepoURL != a.RepoURL {
		t.Fatalf("update: %+v %v", got, err)
	}
	bad := "nope"
	var ve *domain.ValidationError
	if _, err := s.Update(ctx, a.ID, UpdateInput{RepoURL: &bad}); !errors.As(err, &ve) {
		t.Fatalf("bad update: %v", err)
	}
	if _, err := s.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, a.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}
