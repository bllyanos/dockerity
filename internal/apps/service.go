// Package apps holds the app use cases: registering repos and editing them.
package apps

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// DefaultBranch is used when an app is created without one.
const DefaultBranch = "main"

// Store persists apps.
type Store interface {
	ListApps(ctx context.Context) ([]domain.App, error)
	GetApp(ctx context.Context, id int64) (domain.App, error)
	CreateApp(ctx context.Context, a domain.App) (domain.App, error)
	UpdateApp(ctx context.Context, a domain.App) error
	DeleteApp(ctx context.Context, id int64) error
}

// Service implements the app use cases.
type Service struct {
	Store Store
	Now   func() time.Time
}

// CreateInput is what a user provides for a new app.
type CreateInput struct {
	Name          string
	RepoURL       string
	DefaultBranch string
}

// UpdateInput changes the given fields; nil fields are kept.
type UpdateInput struct {
	RepoURL       *string
	DefaultBranch *string
}

// List returns all apps.
func (s *Service) List(ctx context.Context) ([]domain.App, error) {
	return s.Store.ListApps(ctx)
}

// Get returns one app.
func (s *Service) Get(ctx context.Context, id int64) (domain.App, error) {
	a, err := s.Store.GetApp(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.App{}, domain.NotFoundf("app %d not found", id)
	}
	return a, err
}

// Create validates and stores a new app.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.App, error) {
	a := domain.App{
		Name:          strings.TrimSpace(in.Name),
		RepoURL:       strings.TrimSpace(in.RepoURL),
		DefaultBranch: strings.TrimSpace(in.DefaultBranch),
	}
	if a.DefaultBranch == "" {
		a.DefaultBranch = DefaultBranch
	}
	if err := validate(a, true); err != nil {
		return domain.App{}, err
	}
	now := s.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now
	created, err := s.Store.CreateApp(ctx, a)
	if errors.Is(err, domain.ErrConflict) {
		return domain.App{}, domain.Conflictf("an app named %q already exists", a.Name)
	}
	return created, err
}

// Update changes an app's repo URL or default branch.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (domain.App, error) {
	a, err := s.Get(ctx, id)
	if err != nil {
		return domain.App{}, err
	}
	if in.RepoURL != nil {
		a.RepoURL = strings.TrimSpace(*in.RepoURL)
	}
	if in.DefaultBranch != nil {
		a.DefaultBranch = strings.TrimSpace(*in.DefaultBranch)
	}
	if err := validate(a, false); err != nil {
		return domain.App{}, err
	}
	a.UpdatedAt = s.Now().UTC()
	return a, s.Store.UpdateApp(ctx, a)
}

// Delete removes an app.
func (s *Service) Delete(ctx context.Context, id int64) (domain.App, error) {
	a, err := s.Get(ctx, id)
	if err != nil {
		return domain.App{}, err
	}
	return a, s.Store.DeleteApp(ctx, id)
}

func validate(a domain.App, checkName bool) error {
	var errs []error
	if checkName {
		errs = append(errs, domain.ValidateAppName(a.Name))
	}
	errs = append(errs, domain.ValidateRepoURL(a.RepoURL), domain.ValidateBranch(a.DefaultBranch))
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
