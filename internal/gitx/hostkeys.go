// Package gitx is Dockerity's thin layer over go-git (pure Go, so it works in
// the distroless image): SSH auth with deploy keys and host key pinning.
package gitx

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// KnownHost is a pinned SSH host key.
type KnownHost struct {
	Host        string // "github.com:22"
	KeyType     string
	Key         string // base64 wire format
	Fingerprint string // SHA256:...
	AddedAt     time.Time
}

// HostKeyStore persists pinned host keys.
type HostKeyStore interface {
	// GetHostKey returns the pinned key for host or domain.ErrNotFound.
	GetHostKey(ctx context.Context, host string) (KnownHost, error)
	// AddHostKey pins a key; an existing pin for the host is kept.
	AddHostKey(ctx context.Context, k KnownHost) error
}

// HostKeyMismatchError means a host presented a different key than the pinned one.
type HostKeyMismatchError struct {
	Host     string
	Expected string
	Got      string
}

func (e *HostKeyMismatchError) Error() string {
	return fmt.Sprintf("host key for %s changed: pinned %s, got %s; if the change is expected, remove the pinned key", e.Host, e.Expected, e.Got)
}

// TrustOnFirstUse returns a host key callback that pins the first key seen for
// each host and rejects any later key that differs.
func TrustOnFirstUse(ctx context.Context, store HostKeyStore, now func() time.Time) ssh.HostKeyCallback {
	return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
		host := knownhosts.Normalize(hostname)
		got := ssh.FingerprintSHA256(key)
		pinned, err := store.GetHostKey(ctx, host)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return store.AddHostKey(ctx, KnownHost{
				Host: host, KeyType: key.Type(), Key: base64.StdEncoding.EncodeToString(key.Marshal()),
				Fingerprint: got, AddedAt: now().UTC(),
			})
		case err != nil:
			return err
		case pinned.Fingerprint != got:
			return &HostKeyMismatchError{Host: host, Expected: pinned.Fingerprint, Got: got}
		default:
			return nil
		}
	}
}
