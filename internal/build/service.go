package build

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Records persists builds for the service.
type Records interface {
	CreateBuild(ctx context.Context, b domain.Build) (domain.Build, error)
	GetBuild(ctx context.Context, id int64) (domain.Build, error)
	ListBuilds(ctx context.Context, appID int64, limit int) ([]domain.Build, error)
	ListBuildImages(ctx context.Context, buildID int64) ([]domain.BuildImage, error)
	FinishBuild(ctx context.Context, id int64, to domain.BuildStatus, errMsg string, at time.Time) error
}

// Queue runs builds.
type Queue interface {
	Enqueue(buildID int64) error
	Cancel(ctx context.Context, buildID int64) error
}

// Service implements triggering, listing and canceling builds.
type Service struct {
	Builds Records
	Apps   interface {
		Get(ctx context.Context, id int64) (domain.App, error)
	}
	Queue Queue
	Now   func() time.Time
}

// ListLimit is how many builds List returns.
const ListLimit = 50

// Trigger queues a build of ref (the app's default branch when empty).
func (s *Service) Trigger(ctx context.Context, appID int64, ref string, actor *int64) (domain.Build, error) {
	app, err := s.Apps.Get(ctx, appID)
	if err != nil {
		return domain.Build{}, err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = app.DefaultBranch
	}
	if len(ref) > 200 || strings.ContainsAny(ref, " \t\n~^:?*[\\") || strings.HasPrefix(ref, "-") {
		return domain.Build{}, domain.Invalid("ref", "%q is not a valid branch, tag or commit", ref)
	}
	b, err := s.Builds.CreateBuild(ctx, domain.Build{AppID: app.ID, Ref: ref, TriggeredBy: actor, CreatedAt: s.Now().UTC()})
	if err != nil {
		return domain.Build{}, err
	}
	if err := s.Queue.Enqueue(b.ID); err != nil {
		_ = s.Builds.FinishBuild(ctx, b.ID, domain.BuildFailed, err.Error(), s.Now().UTC())
		if errors.Is(err, ErrQueueFull) {
			return domain.Build{}, domain.Conflictf("%s", err)
		}
		return domain.Build{}, err
	}
	return b, nil
}

// List returns the app's recent builds, newest first.
func (s *Service) List(ctx context.Context, appID int64) ([]domain.Build, error) {
	if _, err := s.Apps.Get(ctx, appID); err != nil {
		return nil, err
	}
	return s.Builds.ListBuilds(ctx, appID, ListLimit)
}

// Get returns a build and its images.
func (s *Service) Get(ctx context.Context, id int64) (domain.Build, []domain.BuildImage, error) {
	b, err := s.Builds.GetBuild(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Build{}, nil, domain.NotFoundf("build %d not found", id)
	}
	if err != nil {
		return domain.Build{}, nil, err
	}
	imgs, err := s.Builds.ListBuildImages(ctx, id)
	return b, imgs, err
}

// Cancel stops a queued or running build.
func (s *Service) Cancel(ctx context.Context, id int64) (domain.Build, error) {
	b, _, err := s.Get(ctx, id)
	if err != nil {
		return domain.Build{}, err
	}
	if b.Status.Terminal() {
		return domain.Build{}, domain.Conflictf("build %d already %s", id, b.Status)
	}
	return b, s.Queue.Cancel(ctx, id)
}
