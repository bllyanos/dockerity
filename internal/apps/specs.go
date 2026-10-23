package apps

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
	"github.com/bllyanos/dockerity/internal/textdiff"
)

// MaxSpecBytes bounds the size of a saved spec.
const MaxSpecBytes = 256 * 1024

// SpecStore persists immutable spec versions.
type SpecStore interface {
	InsertSpecVersion(ctx context.Context, v domain.AppSpecVersion) error
	LatestSpecVersion(ctx context.Context, appID int64) (domain.AppSpecVersion, error)
	GetSpecVersion(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error)
	ListSpecVersions(ctx context.Context, appID int64) ([]domain.AppSpecVersion, error)
}

// Specs implements saving and reading app spec versions.
type Specs struct {
	Apps interface {
		Get(ctx context.Context, id int64) (domain.App, error)
	}
	Store SpecStore
	Now   func() time.Time
}

// Save validates yaml and stores it as a new version. If it is identical to
// the latest version, that version is returned with created=false.
// Invalid specs return spec.Problems.
func (s *Specs) Save(ctx context.Context, appID int64, yaml string, actor *int64) (v domain.AppSpecVersion, created bool, err error) {
	app, err := s.Apps.Get(ctx, appID)
	if err != nil {
		return domain.AppSpecVersion{}, false, err
	}
	if len(yaml) > MaxSpecBytes {
		return domain.AppSpecVersion{}, false, domain.Invalid("yaml", "spec is larger than %d KiB", MaxSpecBytes/1024)
	}
	parsed, err := spec.Load([]byte(yaml))
	if err != nil {
		return domain.AppSpecVersion{}, false, err
	}
	if parsed.Name != "" && parsed.Name != app.Name {
		return domain.AppSpecVersion{}, false, spec.Problems{{Pos: parsed.NamePos, Message: "name " + strconv.Quote(parsed.Name) + " does not match the app name " + strconv.Quote(app.Name)}}
	}

	next := 1
	latest, err := s.Store.LatestSpecVersion(ctx, appID)
	switch {
	case err == nil:
		if latest.YAML == yaml {
			return latest, false, nil
		}
		next = latest.Version + 1
	case !errors.Is(err, domain.ErrNotFound):
		return domain.AppSpecVersion{}, false, err
	}

	v = domain.AppSpecVersion{AppID: appID, Version: next, YAML: yaml, CreatedAt: s.Now().UTC(), CreatedBy: actor}
	if err := s.Store.InsertSpecVersion(ctx, v); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return domain.AppSpecVersion{}, false, domain.Conflictf("the spec was changed concurrently, reload and retry")
		}
		return domain.AppSpecVersion{}, false, err
	}
	return v, true, nil
}

// Latest returns the newest version.
func (s *Specs) Latest(ctx context.Context, appID int64) (domain.AppSpecVersion, error) {
	v, err := s.Store.LatestSpecVersion(ctx, appID)
	if errors.Is(err, domain.ErrNotFound) {
		return v, domain.NotFoundf("app %d has no spec yet", appID)
	}
	return v, err
}

// Get returns one version.
func (s *Specs) Get(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error) {
	v, err := s.Store.GetSpecVersion(ctx, appID, version)
	if errors.Is(err, domain.ErrNotFound) {
		return v, domain.NotFoundf("spec version %d not found", version)
	}
	return v, err
}

// List returns version metadata, newest first.
func (s *Specs) List(ctx context.Context, appID int64) ([]domain.AppSpecVersion, error) {
	if _, err := s.Apps.Get(ctx, appID); err != nil {
		return nil, err
	}
	return s.Store.ListSpecVersions(ctx, appID)
}

// Diff returns a unified diff from one version to another.
func (s *Specs) Diff(ctx context.Context, appID int64, from, to int) (string, error) {
	a, err := s.Get(ctx, appID, from)
	if err != nil {
		return "", err
	}
	b, err := s.Get(ctx, appID, to)
	if err != nil {
		return "", err
	}
	return textdiff.Unified("v"+strconv.Itoa(from), "v"+strconv.Itoa(to), a.YAML, b.YAML, 3), nil
}
