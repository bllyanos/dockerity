// Package crypto holds the secret store's primitives: 256-bit keys and
// AES-256-GCM from the Go standard library. Nothing here is custom crypto.
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// KeySize is the AES-256 key length in bytes.
const KeySize = 32

// Key is a 256-bit symmetric key.
type Key [KeySize]byte

// NewKey returns a random key.
func NewKey() (Key, error) {
	var k Key
	_, err := rand.Read(k[:])
	return k, err
}

// ID returns a short, stable fingerprint of the key. It identifies which
// master key wrapped a data key without revealing the key.
func (k Key) ID() string {
	sum := sha256.Sum256(append([]byte("dockerity-key-id:"), k[:]...))
	return hex.EncodeToString(sum[:8])
}

// String hides the key material if a Key is ever printed or logged.
func (k Key) String() string { return "Key(" + k.ID() + ")" }

// GoString hides the key material from %#v.
func (k Key) GoString() string { return k.String() }

// Encode returns the base64 form used in master key files.
func (k Key) Encode() string { return base64.StdEncoding.EncodeToString(k[:]) }

// ParseKey decodes a base64-encoded 32-byte key. Surrounding whitespace is ignored.
func ParseKey(s string) (Key, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Key{}, errors.New("key is empty")
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Key{}, errors.New("key is not valid base64")
	}
	if len(raw) != KeySize {
		return Key{}, fmt.Errorf("key must decode to %d bytes, got %d", KeySize, len(raw))
	}
	var k Key
	copy(k[:], raw)
	return k, nil
}

// ReadKeyFile reads and parses a key file such as /run/secrets/master_key.
// Errors never include the file content.
func ReadKeyFile(path string) (Key, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Key{}, fmt.Errorf("read key file: %w", err)
	}
	k, err := ParseKey(string(data))
	if err != nil {
		return Key{}, fmt.Errorf("key file %s: %w (generate one with: openssl rand -base64 32)", path, err)
	}
	return k, nil
}
