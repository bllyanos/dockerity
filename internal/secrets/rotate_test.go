package secrets

import (
	"context"
	"errors"
	"testing"
	"time"
)

func (m *memKeys) ListDataKeys(context.Context) ([]WrappedKey, error) {
	var out []WrappedKey
	for _, k := range m.keys {
		out = append(out, k)
	}
	return out, nil
}

func (m *memKeys) ReplaceDataKeys(_ context.Context, keys []WrappedKey) error {
	for _, k := range keys {
		m.keys[k.Scope] = k
	}
	return nil
}

func TestRotate(t *testing.T) {
	store := newMemKeys()
	oldMaster, newMaster := newMaster(t), newMaster(t)
	ctx := context.Background()
	before := map[Scope][32]byte{}
	for _, s := range []Scope{EnvScope(1), EnvScope(2), AppScope(1)} {
		k, err := NewKeyring(oldMaster, store, time.Now).DataKey(ctx, s)
		if err != nil {
			t.Fatal(err)
		}
		before[s] = k
	}

	n, err := Rotate(ctx, store, oldMaster, newMaster)
	if err != nil || n != 3 {
		t.Fatalf("rotate: %d %v", n, err)
	}

	ring := NewKeyring(newMaster, store, time.Now)
	if err := ring.Verify(ctx); err != nil {
		t.Fatalf("verify with new key: %v", err)
	}
	for s, want := range before {
		got, err := ring.DataKey(ctx, s)
		if err != nil || got != want {
			t.Fatalf("%s: data key changed or unreadable: %v", s, err)
		}
	}
	if err := NewKeyring(oldMaster, store, time.Now).Verify(ctx); !errors.Is(err, ErrWrongMasterKey) {
		t.Fatalf("old key still accepted: %v", err)
	}
}

func TestRotateRefuses(t *testing.T) {
	store := newMemKeys()
	ctx := context.Background()
	a, b := newMaster(t), newMaster(t)
	if _, err := NewKeyring(a, store, time.Now).DataKey(ctx, EnvScope(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := Rotate(ctx, store, b, newMaster(t)); !errors.Is(err, ErrWrongMasterKey) {
		t.Fatalf("wrong current key: %v", err)
	}
	if _, err := Rotate(ctx, store, a, a); err == nil {
		t.Fatal("rotating to the same key must fail")
	}
}
