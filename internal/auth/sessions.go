package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/auth/token"
	"github.com/bllyanos/dockerity/internal/domain"
)

// UserLookup finds users for authentication.
type UserLookup interface {
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (domain.User, error)
}

// SessionStore persists sessions.
type SessionStore interface {
	CreateSession(ctx context.Context, s domain.Session) error
	GetSession(ctx context.Context, digest string) (domain.Session, error)
	TouchSession(ctx context.Context, digest string, lastSeen, expires time.Time) error
	DeleteSession(ctx context.Context, digest string) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) error
}

// DefaultSessionTTL is how long an idle session stays valid.
const DefaultSessionTTL = 7 * 24 * time.Hour

// touchInterval limits how often activity is written back to the store.
const touchInterval = time.Minute

// Sessions handles password login and cookie sessions.
type Sessions struct {
	Users          UserLookup
	Store          SessionStore
	PasswordParams password.Params
	TTL            time.Duration
	Now            func() time.Time

	dummyOnce sync.Once
	dummyHash string
}

// Login checks credentials and starts a session. Unknown users, wrong
// passwords and disabled users all return domain.ErrUnauthorized.
func (s *Sessions) Login(ctx context.Context, username, pw string) (raw string, sess domain.Session, u domain.User, err error) {
	u, err = s.Users.GetUserByUsername(ctx, domain.NormalizeUsername(username))
	if errors.Is(err, domain.ErrNotFound) {
		// Spend the same time as a real check so usernames cannot be probed by timing.
		_ = password.Verify(pw, s.dummy())
		return "", domain.Session{}, domain.User{}, domain.ErrUnauthorized
	}
	if err != nil {
		return "", domain.Session{}, domain.User{}, err
	}
	if err := password.Verify(pw, u.PasswordHash); err != nil {
		if errors.Is(err, password.ErrMismatch) {
			return "", domain.Session{}, domain.User{}, domain.ErrUnauthorized
		}
		return "", domain.Session{}, domain.User{}, err
	}
	if u.Disabled() {
		return "", domain.Session{}, domain.User{}, domain.ErrUnauthorized
	}
	raw, sess, err = s.Start(ctx, u)
	return raw, sess, u, err
}

// Start creates a session for an already authenticated user and returns its raw token.
func (s *Sessions) Start(ctx context.Context, u domain.User) (string, domain.Session, error) {
	now := s.Now().UTC()
	// Opportunistic cleanup keeps the table small without a background job.
	if err := s.Store.DeleteExpiredSessions(ctx, now); err != nil {
		return "", domain.Session{}, err
	}
	raw, err := token.New(token.PrefixSession)
	if err != nil {
		return "", domain.Session{}, err
	}
	sess := domain.Session{
		TokenDigest: token.Digest(raw),
		UserID:      u.ID,
		CreatedAt:   now,
		LastSeenAt:  now,
		ExpiresAt:   now.Add(s.ttl()),
	}
	if err := s.Store.CreateSession(ctx, sess); err != nil {
		return "", domain.Session{}, err
	}
	return raw, sess, nil
}

// Authenticate resolves a raw session token to its user, sliding the expiry.
// Invalid, expired and disabled-user sessions return domain.ErrUnauthorized.
func (s *Sessions) Authenticate(ctx context.Context, raw string) (Principal, domain.Session, error) {
	if !token.HasPrefix(raw, token.PrefixSession) {
		return Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	sess, err := s.Store.GetSession(ctx, token.Digest(raw))
	if errors.Is(err, domain.ErrNotFound) {
		return Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	if err != nil {
		return Principal{}, domain.Session{}, err
	}
	now := s.Now().UTC()
	if sess.Expired(now) {
		return Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	u, err := s.Users.GetUserByID(ctx, sess.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	if err != nil {
		return Principal{}, domain.Session{}, err
	}
	if u.Disabled() {
		return Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	if now.Sub(sess.LastSeenAt) >= touchInterval {
		sess.LastSeenAt, sess.ExpiresAt = now, now.Add(s.ttl())
		if err := s.Store.TouchSession(ctx, sess.TokenDigest, sess.LastSeenAt, sess.ExpiresAt); err != nil {
			return Principal{}, domain.Session{}, err
		}
	}
	return Principal{User: u}, sess, nil
}

// Logout ends the session for raw. Unknown tokens are ignored.
func (s *Sessions) Logout(ctx context.Context, raw string) error {
	if !token.HasPrefix(raw, token.PrefixSession) {
		return nil
	}
	return s.Store.DeleteSession(ctx, token.Digest(raw))
}

func (s *Sessions) ttl() time.Duration {
	if s.TTL > 0 {
		return s.TTL
	}
	return DefaultSessionTTL
}

func (s *Sessions) dummy() string {
	s.dummyOnce.Do(func() {
		s.dummyHash, _ = password.Hash("dockerity-dummy-password", s.PasswordParams)
	})
	return s.dummyHash
}
