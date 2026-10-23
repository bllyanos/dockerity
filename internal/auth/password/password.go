// Package password hashes and verifies user passwords with argon2id.
// Hashes are PHC strings, so parameters can be raised without breaking old hashes.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Params are argon2id cost parameters.
type Params struct {
	Memory  uint32 // KiB
	Time    uint32
	Threads uint8
	SaltLen uint32
	KeyLen  uint32
}

// DefaultParams follow the OWASP recommendation for argon2id with headroom.
var DefaultParams = Params{Memory: 64 * 1024, Time: 3, Threads: 2, SaltLen: 16, KeyLen: 32}

// MinLength is the minimum password length in characters.
const MinLength = 12

var (
	// ErrMismatch means the password does not match the hash.
	ErrMismatch = errors.New("password does not match")
	// ErrMalformedHash means the stored hash cannot be parsed.
	ErrMalformedHash = errors.New("malformed password hash")
)

var b64 = base64.RawStdEncoding

// Validate checks a new password against the policy.
func Validate(pw string) error {
	if utf8.RuneCountInString(pw) < MinLength {
		return fmt.Errorf("password must be at least %d characters", MinLength)
	}
	return nil
}

// Hash returns the PHC-encoded argon2id hash of pw.
func Hash(pw string, p Params) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify returns nil when pw matches encoded, ErrMismatch when it does not,
// and ErrMalformedHash when encoded cannot be parsed.
func Verify(pw, encoded string) error {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return err
	}
	got := argon2.IDKey([]byte(pw), salt, p.Time, p.Memory, p.Threads, uint32(len(key)))
	if subtle.ConstantTimeCompare(got, key) != 1 {
		return ErrMismatch
	}
	return nil
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, key
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, ErrMalformedHash
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil ||
		p.Memory == 0 || p.Time == 0 || p.Threads == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	return p, salt, key, nil
}
