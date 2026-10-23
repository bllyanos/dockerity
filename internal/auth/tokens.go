package auth

import (
	"context"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/token"
	"github.com/bllyanos/dockerity/internal/domain"
)

// APITokenStore persists API tokens.
type APITokenStore interface {
	CreateAPIToken(ctx context.Context, t domain.APIToken) (domain.APIToken, error)
	ListAPITokens(ctx context.Context, userID int64) ([]domain.APIToken, error)
	GetAPITokenByDigest(ctx context.Context, digest string) (domain.APIToken, error)
	RevokeAPIToken(ctx context.Context, userID, id int64, at time.Time) (domain.APIToken, error)
	TouchAPIToken(ctx context.Context, id int64, at time.Time) error
}

// tokenUseInterval throttles last-use writes and "token used" audit events.
const tokenUseInterval = time.Minute

// Tokens manages API tokens for automation.
type Tokens struct {
	Users UserLookup
	Store APITokenStore
	Now   func() time.Time
}

// Create issues a token for u. The raw value is returned once and never stored.
func (s *Tokens) Create(ctx context.Context, u domain.User, name string) (string, domain.APIToken, error) {
	name = domain.NormalizeTokenName(name)
	if err := domain.ValidateTokenName(name); err != nil {
		return "", domain.APIToken{}, err
	}
	raw, err := token.New(token.PrefixAPI)
	if err != nil {
		return "", domain.APIToken{}, err
	}
	t, err := s.Store.CreateAPIToken(ctx, domain.APIToken{
		UserID:      u.ID,
		Name:        name,
		TokenDigest: token.Digest(raw),
		CreatedAt:   s.Now().UTC(),
	})
	return raw, t, err
}

// List returns the user's tokens.
func (s *Tokens) List(ctx context.Context, userID int64) ([]domain.APIToken, error) {
	return s.Store.ListAPITokens(ctx, userID)
}

// Revoke revokes one of the user's tokens.
func (s *Tokens) Revoke(ctx context.Context, userID, id int64) (domain.APIToken, error) {
	t, err := s.Store.RevokeAPIToken(ctx, userID, id, s.Now().UTC())
	if errors.Is(err, domain.ErrNotFound) {
		return domain.APIToken{}, errors.Join(errors.New("token not found or already revoked"), domain.ErrNotFound)
	}
	return t, err
}

// Authenticate resolves a raw API token to its owner. firstUse is true when
// the last recorded use is older than tokenUseInterval, which is when the
// caller should write a "token used" audit event. Unknown, revoked and
// disabled-user tokens return domain.ErrUnauthorized.
func (s *Tokens) Authenticate(ctx context.Context, raw string) (p Principal, firstUse bool, err error) {
	if !token.HasPrefix(raw, token.PrefixAPI) {
		return Principal{}, false, domain.ErrUnauthorized
	}
	t, err := s.Store.GetAPITokenByDigest(ctx, token.Digest(raw))
	if errors.Is(err, domain.ErrNotFound) {
		return Principal{}, false, domain.ErrUnauthorized
	}
	if err != nil {
		return Principal{}, false, err
	}
	if t.Revoked() {
		return Principal{}, false, domain.ErrUnauthorized
	}
	u, err := s.Users.GetUserByID(ctx, t.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return Principal{}, false, domain.ErrUnauthorized
	}
	if err != nil {
		return Principal{}, false, err
	}
	if u.Disabled() {
		return Principal{}, false, domain.ErrUnauthorized
	}
	now := s.Now().UTC()
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) >= tokenUseInterval {
		if err := s.Store.TouchAPIToken(ctx, t.ID, now); err != nil {
			return Principal{}, false, err
		}
		firstUse = true
	}
	return Principal{User: u, APITokenID: t.ID}, firstUse, nil
}
