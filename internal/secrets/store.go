package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

// Version describes one immutable secret set version. It holds key names,
// never values.
type Version struct {
	Scope     Scope
	Version   int
	Keys      []string
	CreatedAt time.Time
	CreatedBy *int64
}

// SecretStore keeps versioned secret sets per scope. Implementations other
// than Builtin (OpenBao, Infisical) can be added behind this interface.
type SecretStore interface {
	// Update applies changes to the latest version and stores the result as a new version.
	Update(ctx context.Context, scope Scope, changes Changes, actor *int64) (Version, error)
	// Latest returns the newest version, or domain.ErrNotFound if the scope has none.
	Latest(ctx context.Context, scope Scope) (Version, error)
	// Versions lists all versions, newest first.
	Versions(ctx context.Context, scope Scope) ([]Version, error)
	// Values decrypts a version for runtime injection. Never return it to API clients.
	Values(ctx context.Context, scope Scope, version int) (map[string]string, error)
}

// EncryptedVersion is a stored secret set version.
type EncryptedVersion struct {
	Version
	Sealed crypto.Sealed
}

// VersionStore persists encrypted secret set versions.
type VersionStore interface {
	LatestSecretVersion(ctx context.Context, scope Scope) (EncryptedVersion, error)
	GetSecretVersion(ctx context.Context, scope Scope, version int) (EncryptedVersion, error)
	ListSecretVersions(ctx context.Context, scope Scope) ([]Version, error)
	// InsertSecretVersion returns domain.ErrConflict if the version number is taken.
	InsertSecretVersion(ctx context.Context, v EncryptedVersion) error
}

// Builtin is the SecretStore backed by SQLite and envelope encryption.
type Builtin struct {
	Keys  *Keyring
	Store VersionStore
	Now   func() time.Time
}

var _ SecretStore = (*Builtin)(nil)

// Update implements SecretStore.
func (b *Builtin) Update(ctx context.Context, scope Scope, changes Changes, actor *int64) (Version, error) {
	key, err := b.Keys.DataKey(ctx, scope)
	if err != nil {
		return Version{}, err
	}
	current, latest := map[string]string{}, 0
	prev, err := b.Store.LatestSecretVersion(ctx, scope)
	switch {
	case err == nil:
		latest = prev.Version.Version
		if current, err = open(key, prev); err != nil {
			return Version{}, err
		}
	case !errors.Is(err, domain.ErrNotFound):
		return Version{}, err
	}

	next, err := changes.Apply(current)
	if err != nil {
		return Version{}, err
	}
	v := Version{Scope: scope, Version: latest + 1, Keys: SortedKeys(next), CreatedAt: b.Now().UTC(), CreatedBy: actor}
	plaintext, err := json.Marshal(next)
	if err != nil {
		return Version{}, err
	}
	sealed, err := crypto.Seal(key, plaintext, versionAAD(scope, v.Version))
	if err != nil {
		return Version{}, err
	}
	err = b.Store.InsertSecretVersion(ctx, EncryptedVersion{Version: v, Sealed: sealed})
	if errors.Is(err, domain.ErrConflict) {
		return Version{}, domain.Conflictf("secrets were changed concurrently, retry")
	}
	return v, err
}

// Latest implements SecretStore.
func (b *Builtin) Latest(ctx context.Context, scope Scope) (Version, error) {
	v, err := b.Store.LatestSecretVersion(ctx, scope)
	return v.Version, err
}

// Versions implements SecretStore.
func (b *Builtin) Versions(ctx context.Context, scope Scope) ([]Version, error) {
	return b.Store.ListSecretVersions(ctx, scope)
}

// Values implements SecretStore.
func (b *Builtin) Values(ctx context.Context, scope Scope, version int) (map[string]string, error) {
	ev, err := b.Store.GetSecretVersion(ctx, scope, version)
	if err != nil {
		return nil, err
	}
	key, err := b.Keys.DataKey(ctx, scope)
	if err != nil {
		return nil, err
	}
	return open(key, ev)
}

func open(key crypto.Key, ev EncryptedVersion) (map[string]string, error) {
	plaintext, err := crypto.Open(key, ev.Sealed, versionAAD(ev.Scope, ev.Version.Version))
	if err != nil {
		return nil, fmt.Errorf("decrypt secrets %s v%d: %w", ev.Scope, ev.Version.Version, err)
	}
	values := map[string]string{}
	if err := json.Unmarshal(plaintext, &values); err != nil {
		return nil, fmt.Errorf("decode secrets %s v%d: %w", ev.Scope, ev.Version.Version, err)
	}
	return values, nil
}

// versionAAD binds a ciphertext to its scope and version, so rows cannot be swapped.
func versionAAD(scope Scope, version int) []byte {
	return []byte("dockerity/secret-set/" + string(scope) + "/v" + strconv.Itoa(version))
}
