package secrets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

type memKeys struct {
	keys    map[Scope]WrappedKey
	inserts int
}

func newMemKeys() *memKeys { return &memKeys{keys: map[Scope]WrappedKey{}} }

func (m *memKeys) GetDataKey(_ context.Context, s Scope) (WrappedKey, error) {
	k, ok := m.keys[s]
	if !ok {
		return WrappedKey{}, domain.ErrNotFound
	}
	return k, nil
}

func (m *memKeys) InsertDataKey(_ context.Context, k WrappedKey) error {
	m.inserts++
	if _, ok := m.keys[k.Scope]; !ok {
		m.keys[k.Scope] = k
	}
	return nil
}

func (m *memKeys) CountDataKeysNotWrappedBy(_ context.Context, id string) (int, error) {
	n := 0
	for _, k := range m.keys {
		if k.MasterKeyID != id {
			n++
		}
	}
	return n, nil
}

func newMaster(t *testing.T) crypto.Key {
	t.Helper()
	k, err := crypto.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestKeyringCreatesAndReuses(t *testing.T) {
	store := newMemKeys()
	master := newMaster(t)
	ring := NewKeyring(master, store, time.Now)
	ctx := context.Background()

	a, err := ring.DataKey(ctx, EnvScope(1))
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := ring.DataKey(ctx, EnvScope(1)); again != a {
		t.Fatal("same scope must return the same key")
	}
	if b, _ := ring.DataKey(ctx, AppScope(1)); b == a {
		t.Fatal("different scopes must have different keys")
	}
	if store.inserts != 2 {
		t.Fatalf("inserts = %d, want 2", store.inserts)
	}

	// A fresh keyring with the same master key unwraps the stored key.
	if fresh, err := NewKeyring(master, store, time.Now).DataKey(ctx, EnvScope(1)); err != nil || fresh != a {
		t.Fatalf("reload: %v", err)
	}
	if err := ring.Verify(ctx); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestKeyringWrongMasterKey(t *testing.T) {
	store := newMemKeys()
	ctx := context.Background()
	if _, err := NewKeyring(newMaster(t), store, time.Now).DataKey(ctx, EnvScope(1)); err != nil {
		t.Fatal(err)
	}
	other := NewKeyring(newMaster(t), store, time.Now)
	if err := other.Verify(ctx); !errors.Is(err, ErrWrongMasterKey) {
		t.Fatalf("verify: %v", err)
	}
	if _, err := other.DataKey(ctx, EnvScope(1)); !errors.Is(err, ErrWrongMasterKey) {
		t.Fatalf("data key: %v", err)
	}
}

func TestKeyringRejectsSwappedScope(t *testing.T) {
	store := newMemKeys()
	master := newMaster(t)
	ctx := context.Background()
	if _, err := NewKeyring(master, store, time.Now).DataKey(ctx, EnvScope(1)); err != nil {
		t.Fatal(err)
	}
	// Copy env:1's wrapped key onto env:2: the scope-bound AAD must reject it.
	moved := store.keys[EnvScope(1)]
	moved.Scope = EnvScope(2)
	store.keys[EnvScope(2)] = moved
	if _, err := NewKeyring(master, store, time.Now).DataKey(ctx, EnvScope(2)); !errors.Is(err, crypto.ErrDecrypt) {
		t.Fatalf("got %v, want ErrDecrypt", err)
	}
}
