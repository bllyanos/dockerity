package auth

import (
	"context"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/domain"
)

// UserAdminStore is the persistence user management needs.
type UserAdminStore interface {
	ListUsers(ctx context.Context) ([]domain.User, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	SetUserDisabled(ctx context.Context, id int64, at *time.Time) error
	SetUserPassword(ctx context.Context, id int64, hash string) error
}

// CredentialRevoker ends a user's sessions and API tokens.
type CredentialRevoker interface {
	DeleteUserSessions(ctx context.Context, userID int64) error
	RevokeUserAPITokens(ctx context.Context, userID int64, at time.Time) error
}

// UserAdmin implements admin-only user management.
type UserAdmin struct {
	Users          UserAdminStore
	Credentials    CredentialRevoker
	PasswordParams password.Params
	Now            func() time.Time
}

// List returns all users.
func (s *UserAdmin) List(ctx context.Context) ([]domain.User, error) {
	return s.Users.ListUsers(ctx)
}

// Create adds a user. A taken username returns domain.ErrConflict.
func (s *UserAdmin) Create(ctx context.Context, username, pw string, isAdmin bool) (domain.User, error) {
	username = domain.NormalizeUsername(username)
	if err := domain.ValidateUsername(username); err != nil {
		return domain.User{}, err
	}
	hash, err := s.hash(pw)
	if err != nil {
		return domain.User{}, err
	}
	u, err := s.Users.CreateUser(ctx, domain.User{
		Username: username, PasswordHash: hash, IsAdmin: isAdmin, CreatedAt: s.Now().UTC(),
	})
	if errors.Is(err, domain.ErrConflict) {
		return domain.User{}, domain.Conflictf("username %q is taken", username)
	}
	return u, err
}

// SetDisabled disables or re-enables a user. Disabling ends all of the user's
// sessions and revokes their API tokens. Admins cannot disable themselves.
func (s *UserAdmin) SetDisabled(ctx context.Context, actor domain.User, id int64, disabled bool) (domain.User, error) {
	if disabled && actor.ID == id {
		return domain.User{}, domain.Invalid("id", "you cannot disable your own account")
	}
	if _, err := s.get(ctx, id); err != nil {
		return domain.User{}, err
	}
	now := s.Now().UTC()
	var at *time.Time
	if disabled {
		at = &now
	}
	if err := s.Users.SetUserDisabled(ctx, id, at); err != nil {
		return domain.User{}, err
	}
	if disabled {
		if err := s.revoke(ctx, id, now); err != nil {
			return domain.User{}, err
		}
	}
	return s.get(ctx, id)
}

// ResetPassword sets a new password and ends the user's sessions. API tokens are kept.
func (s *UserAdmin) ResetPassword(ctx context.Context, id int64, pw string) (domain.User, error) {
	u, err := s.get(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	hash, err := s.hash(pw)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.Users.SetUserPassword(ctx, id, hash); err != nil {
		return domain.User{}, err
	}
	if err := s.Credentials.DeleteUserSessions(ctx, id); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func (s *UserAdmin) get(ctx context.Context, id int64) (domain.User, error) {
	u, err := s.Users.GetUserByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, domain.NotFoundf("user %d not found", id)
	}
	return u, err
}

func (s *UserAdmin) hash(pw string) (string, error) {
	if err := password.Validate(pw); err != nil {
		return "", domain.Invalid("password", "%s", err)
	}
	return password.Hash(pw, s.PasswordParams)
}

func (s *UserAdmin) revoke(ctx context.Context, userID int64, at time.Time) error {
	if err := s.Credentials.DeleteUserSessions(ctx, userID); err != nil {
		return err
	}
	return s.Credentials.RevokeUserAPITokens(ctx, userID, at)
}
