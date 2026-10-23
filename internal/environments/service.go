// Package environments holds the environment use cases.
package environments

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Store persists environments.
type Store interface {
	ListEnvironments(ctx context.Context, appID int64) ([]domain.Environment, error)
	GetEnvironment(ctx context.Context, id int64) (domain.Environment, error)
	CreateEnvironment(ctx context.Context, e domain.Environment) (domain.Environment, error)
	UpdateEnvironment(ctx context.Context, e domain.Environment) error
	DeleteEnvironment(ctx context.Context, id int64) error
}

// Service implements environment create, update and delete.
type Service struct {
	Store Store
	Apps  interface {
		Get(ctx context.Context, id int64) (domain.App, error)
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

// UpdateDomain changes the environment's domain; an empty domain unroutes it.
func (s *Service) UpdateDomain(ctx context.Context, id int64, host string) (domain.Environment, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return domain.Environment{}, err
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
	return e, nil
}

// Delete removes an environment record.
func (s *Service) Delete(ctx context.Context, id int64) (domain.Environment, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return domain.Environment{}, err
	}
	return e, s.Store.DeleteEnvironment(ctx, id)
}

func normalizeDomain(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}
