package domain

import (
	"regexp"
	"strings"
	"time"
)

// User is a local Dockerity account.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	IsAdmin      bool
	DisabledAt   *time.Time
	CreatedAt    time.Time
}

// Disabled reports whether the user may no longer sign in.
func (u User) Disabled() bool { return u.DisabledAt != nil }

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,31}$`)

// NormalizeUsername trims and lowercases a username.
func NormalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidateUsername checks an already normalized username.
func ValidateUsername(s string) error {
	if !usernamePattern.MatchString(s) {
		return Invalid("username", "must be 3 to 32 characters: lowercase letters, digits, '.', '_' or '-', starting with a letter")
	}
	return nil
}
