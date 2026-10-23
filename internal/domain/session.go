package domain

import "time"

// Session is a logged-in browser session. Only the digest of its token is stored.
type Session struct {
	TokenDigest string
	UserID      int64
	CreatedAt   time.Time
	ExpiresAt   time.Time
	LastSeenAt  time.Time
}

// Expired reports whether the session is no longer valid at now.
func (s Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}
