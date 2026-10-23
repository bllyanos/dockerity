package cleanup

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type logs struct{ cutoff time.Time }

func (l *logs) Expire(cutoff time.Time) ([]int64, error) {
	l.cutoff = cutoff
	return []int64{1}, nil
}

type apps struct{}

func (apps) List(context.Context) ([]domain.App, error) {
	return []domain.App{{Name: "shop"}}, nil
}

type workspaces struct {
	keep map[string]bool
	idle time.Time
}

func (w *workspaces) Prune(keep map[string]bool, idle time.Time) ([]string, error) {
	w.keep, w.idle = keep, idle
	return nil, errors.New("disk busy")
}

type cache struct{ budget int64 }

func (c *cache) PruneBuildCache(_ context.Context, maxBytes int64) (uint64, error) {
	c.budget = maxBytes
	return 42, nil
}

func TestRunOnce(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	builds, deploys := &logs{}, &logs{}
	ws, bc := &workspaces{}, &cache{}
	c := &Cleaner{
		LogStores: map[string]Logs{"build": builds, "deployment": deploys}, Retention: 30 * 24 * time.Hour,
		Apps: apps{}, Workspaces: ws, BuildCache: bc, BuildCacheBudget: 1 << 30,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now },
	}
	err := c.RunOnce(context.Background())
	if err == nil || err.Error() != "disk busy" {
		t.Fatalf("err %v; a failing task must be reported", err)
	}
	cutoff := now.Add(-30 * 24 * time.Hour)
	if !builds.cutoff.Equal(cutoff) || !deploys.cutoff.Equal(cutoff) {
		t.Fatalf("cutoffs %v %v", builds.cutoff, deploys.cutoff)
	}
	if !ws.keep["shop"] || len(ws.keep) != 1 || !ws.idle.Equal(now.Add(-time.Hour)) {
		t.Fatalf("workspaces %+v", ws)
	}
	if bc.budget != 1<<30 {
		t.Fatalf("cache budget %d (other tasks must still run)", bc.budget)
	}
}

func TestRunOnceDisabled(t *testing.T) {
	builds, bc := &logs{}, &cache{}
	c := &Cleaner{LogStores: map[string]Logs{"build": builds}, BuildCache: bc, Now: time.Now,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !builds.cutoff.IsZero() || bc.budget != 0 {
		t.Fatal("zero retention and budget must skip their tasks")
	}
}
