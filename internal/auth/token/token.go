// Package token generates opaque bearer secrets (session cookies, API tokens).
// Only the SHA-256 digest of a token is stored; the raw value is shown once.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// Prefixes make leaked tokens recognizable and tell token kinds apart.
const (
	PrefixSession = "dkts_"
	PrefixAPI     = "dkta_"
)

const randomBytes = 32

// New returns a fresh random token with the given prefix.
func New(prefix string) (string, error) {
	b := make([]byte, randomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// Digest returns the hex SHA-256 of raw, used as the stored lookup key.
// Tokens carry 256 bits of entropy, so a fast unsalted hash is sufficient.
func Digest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// HasPrefix reports whether raw looks like a token of the given kind.
func HasPrefix(raw, prefix string) bool {
	return strings.HasPrefix(raw, prefix) && len(raw) > len(prefix)
}
