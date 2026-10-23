// Package environments holds the environment use cases.
package environments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/settings"
)

// Store persists environments.
type Store interface {
	ListEnvironments(ctx context.Context, appID int64) ([]domain.Environment, error)
	GetEnvironment(ctx context.Context, id int64) (domain.Environment, error)
	CreateEnvironment(ctx context.Context, e domain.Environment) (domain.Environment, error)
	UpdateEnvironment(ctx context.Context, e domain.Environment) error
	DeleteEnvironment(ctx context.Context, id int64) error
	SetExpiry(ctx context.Context, id int64, expires time.Time) error
}

// Service implements environment create, update and delete.
type Service struct {
	Store Store
	Apps  interface {
		Get(ctx context.Context, id int64) (domain.App, error)
	}
	// Deployments, when set, blocks deleting an environment while it deploys.
	Deployments interface {
		ListDeployments(ctx context.Context, envID int64, limit int) ([]domain.Deployment, error)
	}
	// Runtime, when set, removes the environment's containers, network and volumes on delete.
	Runtime interface {
		Remove(ctx context.Context, app, env string) error
		RemoveVolumes(ctx context.Context, app, env string) error
	}
	// Backups, when set, archives stateful volumes before deletion.
	Backups interface {
		ArchiveEnvironment(ctx context.Context, env domain.Environment) ([]domain.Backup, error)
	}
	// Routes, when set, rewrites the environment's routes after a domain change.
	Routes interface {
		RebuildRoutes(ctx context.Context, envID int64) error
	}
	// Settings provides the base domain of ephemeral environments.
	Settings interface {
		Get(ctx context.Context) (settings.Settings, error)
	}
	Now func() time.Time
}

// List returns an app's environments.
func (s *Service) List(ctx context.Context, appID int64) ([]domain.Environment, error) {
	if _, err := s.Apps.Get(ctx, appID); err != nil {
		return nil, err
	}
	return s.Store.ListEnvironments(ctx, appID)
}

// Get returns one environment.
func (s *Service) Get(ctx context.Context, id int64) (domain.Environment, error) {
	e, err := s.Store.GetEnvironment(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return e, domain.NotFoundf("environment %d not found", id)
	}
	return e, err
}

// CreatePermanent adds a permanent environment to an app.
func (s *Service) CreatePermanent(ctx context.Context, appID int64, name, host string) (domain.Environment, error) {
	if _, err := s.Apps.Get(ctx, appID); err != nil {
		return domain.Environment{}, err
	}
	name, host = strings.TrimSpace(name), normalizeDomain(host)
	if err := domain.ValidateEnvName(name); err != nil {
		return domain.Environment{}, err
	}
	if err := domain.ValidateDomain(host); err != nil {
		return domain.Environment{}, err
	}
	now := s.Now().UTC()
	e, err := s.Store.CreateEnvironment(ctx, domain.Environment{
		AppID: appID, Name: name, Type: domain.EnvPermanent, Domain: host, CreatedAt: now, UpdatedAt: now,
	})
	if errors.Is(err, domain.ErrConflict) {
		return domain.Environment{}, domain.Conflictf("the environment name or domain is already used")
	}
	return e, err
}

// CreateEphemeral adds an ephemeral environment at <name>-<app>.<baseDomain>
// that expires after ttl (DefaultEphemeralTTL when zero).
func (s *Service) CreateEphemeral(ctx context.Context, appID int64, name string, ttl time.Duration) (domain.Environment, error) {
	app, err := s.Apps.Get(ctx, appID)
	if err != nil {
		return domain.Environment{}, err
	}
	name = strings.TrimSpace(name)
	if err := domain.ValidateEnvName(name); err != nil {
		return domain.Environment{}, err
	}
	if ttl == 0 {
		ttl = domain.DefaultEphemeralTTL
	}
	if ttl < domain.MinEphemeralTTL || ttl > domain.MaxEphemeralTTL {
		return domain.Environment{}, domain.Invalid("ttl", "must be between %s and %s", domain.MinEphemeralTTL, domain.MaxEphemeralTTL)
	}
	cfg, err := s.Settings.Get(ctx)
	if err != nil {
		return domain.Environment{}, err
	}
	if cfg.BaseDomain == "" {
		return domain.Environment{}, domain.Invalid("baseDomain", "set a base domain in the settings before spawning ephemeral environments")
	}
	host, err := domain.EphemeralDomain(name, app.Name, cfg.BaseDomain)
	if err != nil {
		return domain.Environment{}, err
	}
	if err := domain.ValidateDomain(host); err != nil {
		return domain.Environment{}, err
	}
	now := s.Now().UTC()
	expires := now.Add(ttl)
	e, err := s.Store.CreateEnvironment(ctx, domain.Environment{
		AppID: appID, Name: name, Type: domain.EnvEphemeral, Domain: host, TTL: ttl, ExpiresAt: &expires,
		CreatedAt: now, UpdatedAt: now,
	})
	if errors.Is(err, domain.ErrConflict) {
		return domain.Environment{}, domain.Conflictf("an environment named %s already exists", name)
	}
	return e, err
}

// Extend pushes an ephemeral environment's expiry to now plus its TTL.
// Permanent environments are left alone.
func (s *Service) Extend(ctx context.Context, id int64) (domain.Environment, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return domain.Environment{}, err
	}
	if e.Type != domain.EnvEphemeral {
		return e, nil
	}
	expires := s.Now().UTC().Add(e.TTL)
	if err := s.Store.SetExpiry(ctx, id, expires); err != nil {
		return domain.Environment{}, err
	}
	e.ExpiresAt = &expires
	return e, nil
}

// UpdateDomain changes the environment's domain; an empty domain unroutes it.
func (s *Service) UpdateDomain(ctx context.Context, id int64, host string) (domain.Environment, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return domain.Environment{}, err
	}
	if e.Type == domain.EnvEphemeral {
		return domain.Environment{}, domain.Invalid("domain", "ephemeral environments use their automatic domain")
	}
	e.Domain = normalizeDomain(host)
	if err := domain.ValidateDomain(e.Domain); err != nil {
		return domain.Environment{}, err
	}
	e.UpdatedAt = s.Now().UTC()
	if err := s.Store.UpdateEnvironment(ctx, e); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return domain.Environment{}, domain.Conflictf("domain %s is used by another environment", e.Domain)
		}
		return domain.Environment{}, err
	}
	if s.Routes != nil {
		if err := s.Routes.RebuildRoutes(ctx, e.ID); err != nil {
			return domain.Environment{}, err
		}
	}
	return e, nil
}

// Delete removes an environment: its routes, containers and network, then a
// backup of its stateful volumes (unless skipBackup), then the volumes and the
// record. It is refused while a deployment is in progress. A failed backup
// stops the deletion before any volume is removed.
func (s *Service) Delete(ctx context.Context, id int64, skipBackup bool) (domain.Environment, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return domain.Environment{}, err
	}
	if s.Deployments != nil {
		latest, err := s.Deployments.ListDeployments(ctx, id, 1)
		if err != nil {
			return domain.Environment{}, err
		}
		if len(latest) > 0 && !latest[0].Status.Terminal() {
			return domain.Environment{}, domain.Conflictf("a deployment is in progress in %s; wait for it to finish", e.Name)
		}
	}
	if s.Runtime != nil {
		if err := s.Runtime.Remove(ctx, e.AppName, e.Name); err != nil {
			return domain.Environment{}, err
		}
	}
	if !skipBackup && s.Backups != nil {
		if _, err := s.Backups.ArchiveEnvironment(ctx, e); err != nil {
			return domain.Environment{}, fmt.Errorf("back up %s before deleting it (its volumes are kept): %w", e.Name, err)
		}
	}
	if s.Runtime != nil {
		if err := s.Runtime.RemoveVolumes(ctx, e.AppName, e.Name); err != nil {
			return domain.Environment{}, err
		}
	}
	return e, s.Store.DeleteEnvironment(ctx, id)
}

func normalizeDomain(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}
