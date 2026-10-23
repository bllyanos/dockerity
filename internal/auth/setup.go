// Package auth holds the authentication use cases: first-run setup, login,
// sessions and API tokens. Persistence is behind small store interfaces.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/domain"
)

// SetupUserStore is the persistence Setup needs.
type SetupUserStore interface {
	CountUsers(ctx context.Context) (int, error)
	// CreateFirstAdmin inserts u only when no user exists, atomically.
	// It returns domain.ErrConflict when a user already exists.
	CreateFirstAdmin(ctx context.Context, u domain.User) (domain.User, error)
}

// Setup creates the first admin on a fresh install.
type Setup struct {
	Users          SetupUserStore
	PasswordParams password.Params
	Now            func() time.Time
}

// Needed reports whether no user exists yet.
func (s *Setup) Needed(ctx context.Context) (bool, error) {
	n, err := s.Users.CountUsers(ctx)
	return n == 0, err
}

// Run creates the first admin. It fails with domain.ErrConflict once any user exists.
func (s *Setup) Run(ctx context.Context, username, pw string) (domain.User, error) {
	username = domain.NormalizeUsername(username)
	if err := domain.ValidateUsername(username); err != nil {
		return domain.User{}, err
	}
	if err := password.Validate(pw); err != nil {
		return domain.User{}, domain.Invalid("password", "%s", err)
	}
	hash, err := password.Hash(pw, s.PasswordParams)
	if err != nil {
		return domain.User{}, err
	}
	u, err := s.Users.CreateFirstAdmin(ctx, domain.User{
		Username:     username,
		PasswordHash: hash,
		IsAdmin:      true,
		CreatedAt:    s.Now().UTC(),
	})
	if errors.Is(err, domain.ErrConflict) {
		return domain.User{}, fmt.Errorf("setup already completed: %w", err)
	}
	return u, err
}
