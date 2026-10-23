package reaper

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type store struct{ envs []domain.Environment }

func (s *store) ListExpired(_ context.Context, now time.Time) ([]domain.Environment, error) {
	var out []domain.Environment
	for _, e := range s.envs {
		if e.Expired(now) {
			out = append(out, e)
		}
	}
	return out, nil
}

type deleter struct {
	busy    map[int64]bool
	deleted []int64
}

func (d *deleter) Delete(_ context.Context, id int64, skipBackup bool) (domain.Environment, error) {
	if skipBackup {
		return domain.Environment{}, errors.New("the reaper must back up")
	}
	if d.busy[id] {
		return domain.Environment{}, domain.Conflictf("deployment in progress")
	}
	d.deleted = append(d.deleted, id)
	return domain.Environment{}, nil
}

type audit struct{ actions []string }

func (a *audit) Record(_ context.Context, e domain.AuditEvent) error {
	a.actions = append(a.actions, e.Action)
	return nil
}

func TestRunOnce(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { t := now.Add(d); return &t }
	st := &store{envs: []domain.Environment{
		{ID: 1, AppName: "shop", Name: "pr-1", ExpiresAt: at(-time.Hour)},
		{ID: 2, AppName: "shop", Name: "pr-2", ExpiresAt: at(time.Hour)},
		{ID: 3, AppName: "shop", Name: "pr-3", ExpiresAt: at(-time.Minute)},
		{ID: 4, AppName: "shop", Name: "prod"},
	}}
	d := &deleter{busy: map[int64]bool{3: true}}
	a := &audit{}
	r := &Reaper{Expired: st, Envs: d, Audit: a, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now }}

	got := r.RunOnce(context.Background())
	if !slices.Equal(got, []string{"shop/pr-1"}) || !slices.Equal(d.deleted, []int64{1}) {
		t.Fatalf("reaped %v deleted %v", got, d.deleted)
	}
	if !slices.Equal(a.actions, []string{domain.AuditEnvironmentReaped}) {
		t.Fatalf("audit %v", a.actions)
	}

	delete(d.busy, 3)
	st.envs = st.envs[1:] // pr-1 is gone now
	if got := r.RunOnce(context.Background()); !slices.Equal(got, []string{"shop/pr-3"}) {
		t.Fatalf("second round %v", got)
	}
}

func TestRunStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Reaper{Expired: &store{}, Envs: &deleter{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now: time.Now, Interval: time.Millisecond}
	done := make(chan struct{})
	go func() { r.Run(ctx); close(done) }()
	time.Sleep(5 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reaper did not stop")
	}
}
