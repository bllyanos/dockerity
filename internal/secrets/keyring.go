package secrets

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

// WrappedKey is a data key encrypted by the master key, as stored.
type WrappedKey struct {
	Scope       Scope
	Sealed      crypto.Sealed
	MasterKeyID string
	CreatedAt   time.Time
}

// DataKeyStore persists wrapped data keys.
type DataKeyStore interface {
	GetDataKey(ctx context.Context, scope Scope) (WrappedKey, error)
	// InsertDataKey stores k unless the scope already has a key. It is not an
	// error if another writer won the race.
	InsertDataKey(ctx context.Context, k WrappedKey) error
	// CountDataKeysNotWrappedBy counts keys wrapped by another master key.
	CountDataKeysNotWrappedBy(ctx context.Context, masterKeyID string) (int, error)
}

// ErrWrongMasterKey means data keys were wrapped by a different master key.
var ErrWrongMasterKey = errors.New("data keys were wrapped by a different master key")

// Keyring hands out unwrapped data keys per scope, creating them on first use.
type Keyring struct {
	master crypto.Key
	store  DataKeyStore
	now    func() time.Time

	mu    sync.Mutex
	cache map[Scope]crypto.Key
}

// NewKeyring returns a keyring using master to wrap data keys.
func NewKeyring(master crypto.Key, store DataKeyStore, now func() time.Time) *Keyring {
	return &Keyring{master: master, store: store, now: now, cache: map[Scope]crypto.Key{}}
}

// Verify fails with ErrWrongMasterKey when any stored data key was wrapped by
// another master key. Run it at startup so a wrong key fails fast.
func (k *Keyring) Verify(ctx context.Context) error {
	n, err := k.store.CountDataKeysNotWrappedBy(ctx, k.master.ID())
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%w: %d data key(s) do not match master key %s", ErrWrongMasterKey, n, k.master.ID())
	}
	return nil
}

// DataKey returns the scope's data key, creating and storing it if needed.
func (k *Keyring) DataKey(ctx context.Context, scope Scope) (crypto.Key, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if key, ok := k.cache[scope]; ok {
		return key, nil
	}

	w, err := k.store.GetDataKey(ctx, scope)
	if errors.Is(err, domain.ErrNotFound) {
		if err := k.create(ctx, scope); err != nil {
			return crypto.Key{}, err
		}
		w, err = k.store.GetDataKey(ctx, scope)
	}
	if err != nil {
		return crypto.Key{}, err
	}
	key, err := k.unwrap(w)
	if err != nil {
		return crypto.Key{}, err
	}
	k.cache[scope] = key
	return key, nil
}

func (k *Keyring) create(ctx context.Context, scope Scope) error {
	key, err := crypto.NewKey()
	if err != nil {
		return err
	}
	sealed, err := crypto.Seal(k.master, key[:], wrapAAD(scope))
	if err != nil {
		return err
	}
	return k.store.InsertDataKey(ctx, WrappedKey{
		Scope: scope, Sealed: sealed, MasterKeyID: k.master.ID(), CreatedAt: k.now().UTC(),
	})
}

func (k *Keyring) unwrap(w WrappedKey) (crypto.Key, error) {
	if w.MasterKeyID != k.master.ID() {
		return crypto.Key{}, fmt.Errorf("%w: scope %s uses %s, loaded %s", ErrWrongMasterKey, w.Scope, w.MasterKeyID, k.master.ID())
	}
	raw, err := crypto.Open(k.master, w.Sealed, wrapAAD(w.Scope))
	if err != nil {
		return crypto.Key{}, fmt.Errorf("unwrap data key for %s: %w", w.Scope, err)
	}
	var key crypto.Key
	if len(raw) != crypto.KeySize {
		return crypto.Key{}, fmt.Errorf("unwrap data key for %s: unexpected length", w.Scope)
	}
	copy(key[:], raw)
	return key, nil
}

// wrapAAD binds a wrapped data key to its scope.
func wrapAAD(scope Scope) []byte {
	return []byte("dockerity/data-key/" + string(scope))
}
