// Package reaper deletes expired ephemeral environments
// (docs/design.md section 7).
package reaper

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// DefaultInterval is how often the reaper looks for expired environments.
const DefaultInterval = time.Minute

// Reaper removes expired ephemeral environments through the normal deletion
// flow, so their volumes are backed up first.
type Reaper struct {
	Expired interface {
		ListExpired(ctx context.Context, now time.Time) ([]domain.Environment, error)
	}
	Envs interface {
		Delete(ctx context.Context, id int64, skipBackup bool) (domain.Environment, error)
	}
	Audit interface {
		Record(ctx context.Context, e domain.AuditEvent) error
	}
	Logger   *slog.Logger
	Now      func() time.Time
	Interval time.Duration
}

// Run reaps on every interval until ctx is cancelled.
func (r *Reaper) Run(ctx context.Context) {
	interval := r.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		r.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce deletes every environment expired by now and returns their names.
// Environments with a deployment in progress are left for the next round.
func (r *Reaper) RunOnce(ctx context.Context) []string {
	now := r.Now().UTC()
	expired, err := r.Expired.ListExpired(ctx, now)
	if err != nil {
		r.Logger.Error("list expired environments", "error", err)
		return nil
	}
	var reaped []string
	for _, e := range expired {
		if _, err := r.Envs.Delete(ctx, e.ID, false); err != nil {
			if errors.Is(err, domain.ErrConflict) {
				r.Logger.Info("expired environment is busy; retrying later", "env", e.AppName+"/"+e.Name)
				continue
			}
			r.Logger.Error("reap environment", "env", e.AppName+"/"+e.Name, "error", err)
			continue
		}
		reaped = append(reaped, e.AppName+"/"+e.Name)
		r.Logger.Info("reaped expired environment", "env", e.AppName+"/"+e.Name)
		if r.Audit != nil {
			_ = r.Audit.Record(ctx, domain.AuditEvent{
				At: now, Action: domain.AuditEnvironmentReaped, TargetType: "environment", TargetID: strconv.FormatInt(e.ID, 10),
				Metadata: map[string]string{"app": e.AppName, "name": e.Name, "expired_at": e.ExpiresAt.Format(time.RFC3339)},
			})
		}
	}
	return reaped
}
