// Package envsecrets exposes an environment's secrets: write-only values,
// readable key names and version history.
package envsecrets

import (
	"context"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// Summary describes the latest secret set of an environment without values.
type Summary struct {
	// Version is 0 when the environment has no secrets yet.
	Version int
	Keys    []string
}

// Service reads and updates environment secrets.
type Service struct {
	Envs interface {
		Get(ctx context.Context, id int64) (domain.Environment, error)
	}
	Secrets secrets.SecretStore
}

// Current returns the key names of the latest version.
func (s *Service) Current(ctx context.Context, envID int64) (Summary, error) {
	if _, err := s.Envs.Get(ctx, envID); err != nil {
		return Summary{}, err
	}
	v, err := s.Secrets.Latest(ctx, secrets.EnvScope(envID))
	if errors.Is(err, domain.ErrNotFound) {
		return Summary{Keys: []string{}}, nil
	}
	if err != nil {
		return Summary{}, err
	}
	return Summary{Version: v.Version, Keys: v.Keys}, nil
}

// Update applies changes as a new version and returns its summary.
func (s *Service) Update(ctx context.Context, envID int64, changes secrets.Changes, actor *int64) (Summary, error) {
	if _, err := s.Envs.Get(ctx, envID); err != nil {
		return Summary{}, err
	}
	v, err := s.Secrets.Update(ctx, secrets.EnvScope(envID), changes, actor)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Version: v.Version, Keys: v.Keys}, nil
}

// Versions lists all versions, newest first.
func (s *Service) Versions(ctx context.Context, envID int64) ([]secrets.Version, error) {
	if _, err := s.Envs.Get(ctx, envID); err != nil {
		return nil, err
	}
	return s.Secrets.Versions(ctx, secrets.EnvScope(envID))
}
