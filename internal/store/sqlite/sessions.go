package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Sessions is the session repository.
type Sessions struct {
	DB *sql.DB
}

// CreateSession stores s.
func (r Sessions) CreateSession(ctx context.Context, s domain.Session) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO sessions (token_digest, user_id, created_at, expires_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?)`,
		s.TokenDigest, s.UserID, formatTime(s.CreatedAt), formatTime(s.ExpiresAt), formatTime(s.LastSeenAt))
	return err
}

// GetSession returns the session with digest or domain.ErrNotFound.
func (r Sessions) GetSession(ctx context.Context, digest string) (domain.Session, error) {
	var (
		s                         domain.Session
		created, expires, lastSee string
	)
	err := r.DB.QueryRowContext(ctx, `
		SELECT token_digest, user_id, created_at, expires_at, last_seen_at
		FROM sessions WHERE token_digest = ?`, digest).
		Scan(&s.TokenDigest, &s.UserID, &created, &expires, &lastSee)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Session{}, err
	}
	for _, f := range []struct {
		dst *time.Time
		src string
	}{{&s.CreatedAt, created}, {&s.ExpiresAt, expires}, {&s.LastSeenAt, lastSee}} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return domain.Session{}, err
		}
	}
	return s, nil
}

// TouchSession records activity and extends the expiry.
func (r Sessions) TouchSession(ctx context.Context, digest string, lastSeen, expires time.Time) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE token_digest = ?`,
		formatTime(lastSeen), formatTime(expires), digest)
	return err
}

// DeleteSession removes one session. Deleting a missing session is not an error.
func (r Sessions) DeleteSession(ctx context.Context, digest string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_digest = ?`, digest)
	return err
}

// DeleteUserSessions removes every session of a user.
func (r Sessions) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// DeleteExpiredSessions removes sessions that expired at or before now.
func (r Sessions) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, formatTime(now))
	return err
}
