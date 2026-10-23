package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Audit actions for API tokens.
const (
	AuditTokenCreated = "token.created"
	AuditTokenRevoked = "token.revoked"
	AuditTokenUsed    = "token.used"
)

// APIToken is a long-lived bearer credential for automation. Only the digest
// of its raw value is stored.
type APIToken struct {
	ID          int64
	UserID      int64
	Name        string
	TokenDigest string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
}

// Revoked reports whether the token can no longer be used.
func (t APIToken) Revoked() bool { return t.RevokedAt != nil }

// NormalizeTokenName trims surrounding space.
func NormalizeTokenName(s string) string { return strings.TrimSpace(s) }

// ValidateTokenName checks an already normalized token name.
func ValidateTokenName(s string) error {
	if n := utf8.RuneCountInString(s); n < 1 || n > 64 {
		return Invalid("name", "must be 1 to 64 characters")
	}
	return nil
}
