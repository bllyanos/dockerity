package build

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type recordsFake struct {
	*memBuilds
}

func (r recordsFake) CreateBuild(_ context.Context, b domain.Build) (domain.Build, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b.ID = int64(len(r.builds) + 1)
	b.Status = domain.BuildQueued
	r.builds[b.ID] = &b
	return b, nil
}

func (r recordsFake) ListBuilds(context.Context, int64, int) ([]domain.Build, error) {
	return r.ListBuildsByStatus(context.Background(), domain.BuildQueued)
}

func (r recordsFake) ListBuildImages(context.Context, int64) ([]domain.BuildImage, error) {
	return r.images, nil
}

type queueFake struct {
	enqueued []int64
	canceled []int64
	full     bool
}

func (q *queueFake) Enqueue(id int64) error {
	if q.full {
		return ErrQueueFull
	}
	q.enqueued = append(q.enqueued, id)
	return nil
}

func (q *queueFake) Cancel(_ context.Context, id int64) error {
	q.canceled = append(q.canceled, id)
	return nil
}

type defaultBranchApps struct{}

func (defaultBranchApps) Get(_ context.Context, id int64) (domain.App, error) {
	if id != 1 {
		return domain.App{}, domain.ErrNotFound
	}
	return domain.App{ID: 1, Name: "shop", DefaultBranch: "main"}, nil
}

func TestServiceTrigger(t *testing.T) {
	records := recordsFake{newMemBuilds()}
	q := &queueFake{}
	s := &Service{Builds: records, Apps: defaultBranchApps{}, Queue: q, Now: time.Now}
	ctx := context.Background()

	b, err := s.Trigger(ctx, 1, "", nil)
	if err != nil || b.Ref != "main" || b.Status != domain.BuildQueued || len(q.enqueued) != 1 {
		t.Fatalf("default branch: %+v %v %v", b, err, q.enqueued)
	}
	if b, _ := s.Trigger(ctx, 1, " v1.2.0 ", nil); b.Ref != "v1.2.0" {
		t.Fatalf("tag ref: %+v", b)
	}
	var ve *domain.ValidationError
	if _, err := s.Trigger(ctx, 1, "bad ref", nil); !errors.As(err, &ve) {
		t.Fatalf("invalid ref: %v", err)
	}
	if _, err := s.Trigger(ctx, 9, "", nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing app: %v", err)
	}

	q.full = true
	if _, err := s.Trigger(ctx, 1, "", nil); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("full queue: %v", err)
	}
	if records.status(3) != domain.BuildFailed {
		t.Fatal("a build that could not be queued must not stay queued")
	}
}

func TestServiceCancel(t *testing.T) {
	done := queued(2)
	done.Status = domain.BuildSucceeded
	records := recordsFake{newMemBuilds(queued(1), done)}
	q := &queueFake{}
	s := &Service{Builds: records, Apps: defaultBranchApps{}, Queue: q, Now: time.Now}
	ctx := context.Background()

	if _, err := s.Cancel(ctx, 1); err != nil || len(q.canceled) != 1 {
		t.Fatalf("cancel: %v %v", err, q.canceled)
	}
	if _, err := s.Cancel(ctx, 2); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("cancel finished: %v", err)
	}
	if _, err := s.Cancel(ctx, 9); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cancel missing: %v", err)
	}
}
