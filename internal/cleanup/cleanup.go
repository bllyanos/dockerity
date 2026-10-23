// Package cleanup runs housekeeping on a schedule: log retention, workspaces
// of deleted apps and the BuildKit cache budget.
package cleanup

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// DefaultInterval is how often cleanup runs.
const DefaultInterval = time.Hour

// workspaceIdle protects clones touched recently, e.g. for an app created
// after the app list was read.
const workspaceIdle = time.Hour

// Logs expires stored logs.
type Logs interface {
	Expire(cutoff time.Time) ([]int64, error)
}

// Cleaner runs the housekeeping tasks.
type Cleaner struct {
	// LogStores hold build and deployment logs, by kind ("build", "deployment").
	LogStores map[string]Logs
	// Retention is how long logs are kept; zero keeps them forever.
	Retention time.Duration
	Apps      interface {
		List(ctx context.Context) ([]domain.App, error)
	}
	Workspaces interface {
		Prune(keep map[string]bool, idle time.Time) ([]string, error)
	}
	BuildCache interface {
		PruneBuildCache(ctx context.Context, maxBytes int64) (uint64, error)
	}
	// BuildCacheBudget is the most bytes the build cache may use; zero skips pruning.
	BuildCacheBudget int64
	Logger           *slog.Logger
	Now              func() time.Time
	Interval         time.Duration
}

// Run cleans up on every interval until ctx is cancelled, starting now.
func (c *Cleaner) Run(ctx context.Context) {
	interval := c.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if err := c.RunOnce(ctx); err != nil {
			c.Logger.Error("cleanup", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce runs every task once; a failing task does not stop the others.
func (c *Cleaner) RunOnce(ctx context.Context) error {
	return errors.Join(c.expireLogs(), c.pruneWorkspaces(ctx), c.pruneBuildCache(ctx))
}

func (c *Cleaner) expireLogs() error {
	if c.Retention <= 0 {
		return nil
	}
	cutoff := c.Now().Add(-c.Retention)
	var errs []error
	for kind, store := range c.LogStores {
		ids, err := store.Expire(cutoff)
		if err != nil {
			errs = append(errs, err)
		}
		if len(ids) > 0 {
			c.Logger.Info("expired logs", "kind", kind, "count", len(ids))
		}
	}
	return errors.Join(errs...)
}

func (c *Cleaner) pruneWorkspaces(ctx context.Context) error {
	if c.Workspaces == nil {
		return nil
	}
	apps, err := c.Apps.List(ctx)
	if err != nil {
		return err
	}
	keep := make(map[string]bool, len(apps))
	for _, a := range apps {
		keep[a.Name] = true
	}
	removed, err := c.Workspaces.Prune(keep, c.Now().Add(-workspaceIdle))
	for _, name := range removed {
		c.Logger.Info("removed workspace of deleted app", "app", name)
	}
	return err
}

func (c *Cleaner) pruneBuildCache(ctx context.Context) error {
	if c.BuildCache == nil || c.BuildCacheBudget <= 0 {
		return nil
	}
	freed, err := c.BuildCache.PruneBuildCache(ctx, c.BuildCacheBudget)
	if err != nil {
		return err
	}
	if freed > 0 {
		c.Logger.Info("pruned build cache", "bytes", freed)
	}
	return nil
}
