package apps

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

type memSpecs struct{ versions []domain.AppSpecVersion }

func (m *memSpecs) InsertSpecVersion(_ context.Context, v domain.AppSpecVersion) error {
	for _, x := range m.versions {
		if x.AppID == v.AppID && x.Version == v.Version {
			return domain.ErrConflict
		}
	}
	m.versions = append(m.versions, v)
	return nil
}

func (m *memSpecs) LatestSpecVersion(_ context.Context, appID int64) (domain.AppSpecVersion, error) {
	var best *domain.AppSpecVersion
	for i, v := range m.versions {
		if v.AppID == appID && (best == nil || v.Version > best.Version) {
			best = &m.versions[i]
		}
	}
	if best == nil {
		return domain.AppSpecVersion{}, domain.ErrNotFound
	}
	return *best, nil
}

func (m *memSpecs) GetSpecVersion(_ context.Context, appID int64, version int) (domain.AppSpecVersion, error) {
	for _, v := range m.versions {
		if v.AppID == appID && v.Version == version {
			return v, nil
		}
	}
	return domain.AppSpecVersion{}, domain.ErrNotFound
}

func (m *memSpecs) ListSpecVersions(_ context.Context, appID int64) ([]domain.AppSpecVersion, error) {
	return m.versions, nil
}

const specV1 = "name: shop\nservices:\n  web:\n    kind: deployment\n    strategy: recreate\n    image: nginx\n"
const specV2 = "name: shop\nservices:\n  web:\n    kind: deployment\n    strategy: recreate\n    image: nginx:1.27\n"

func newSpecs(t *testing.T) (*Specs, int64) {
	t.Helper()
	svc := newService()
	a, err := svc.Create(context.Background(), CreateInput{Name: "shop", RepoURL: "git@github.com:org/shop.git"})
	if err != nil {
		t.Fatal(err)
	}
	return &Specs{Apps: svc, Store: &memSpecs{}, Now: time.Now}, a.ID
}

func TestSpecsSave(t *testing.T) {
	s, appID := newSpecs(t)
	ctx := context.Background()

	v1, created, err := s.Save(ctx, appID, specV1, nil)
	if err != nil || !created || v1.Version != 1 {
		t.Fatalf("v1: %+v %v %v", v1, created, err)
	}
	same, created, err := s.Save(ctx, appID, specV1, nil)
	if err != nil || created || same.Version != 1 {
		t.Fatalf("unchanged save: %+v %v %v", same, created, err)
	}
	v2, created, _ := s.Save(ctx, appID, specV2, nil)
	if !created || v2.Version != 2 {
		t.Fatalf("v2: %+v", v2)
	}
	if old, _ := s.Get(ctx, appID, 1); old.YAML != specV1 {
		t.Fatal("old version modified")
	}

	diff, err := s.Diff(ctx, appID, 1, 2)
	if err != nil || !strings.Contains(diff, "-    image: nginx\n+    image: nginx:1.27\n") {
		t.Fatalf("diff:\n%s %v", diff, err)
	}
}

func TestSpecsSaveRejects(t *testing.T) {
	s, appID := newSpecs(t)
	ctx := context.Background()
	var ps spec.Problems

	_, _, err := s.Save(ctx, appID, "services:\n  web:\n    kind: daemon\n    image: x\n", nil)
	if !errors.As(err, &ps) || ps[0].Pos.Line != 3 {
		t.Fatalf("invalid spec: %v", err)
	}
	_, _, err = s.Save(ctx, appID, strings.Replace(specV1, "name: shop", "name: blog", 1), nil)
	if !errors.As(err, &ps) || !strings.Contains(ps[0].Message, "does not match the app name") {
		t.Fatalf("name mismatch: %v", err)
	}
	if _, _, err := s.Save(ctx, 99, specV1, nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing app: %v", err)
	}
	var ve *domain.ValidationError
	if _, _, err := s.Save(ctx, appID, strings.Repeat("#", MaxSpecBytes+1), nil); !errors.As(err, &ve) {
		t.Fatalf("too large: %v", err)
	}
	if _, err := s.Latest(ctx, appID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("no spec yet: %v", err)
	}
}
