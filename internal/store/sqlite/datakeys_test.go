package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

func TestDataKeysWithKeyring(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	master, _ := crypto.NewKey()
	ring := secrets.NewKeyring(master, DataKeys{DB: db}, time.Now)

	first, err := ring.DataKey(ctx, secrets.EnvScope(7))
	if err != nil {
		t.Fatal(err)
	}
	again, err := secrets.NewKeyring(master, DataKeys{DB: db}, time.Now).DataKey(ctx, secrets.EnvScope(7))
	if err != nil || again != first {
		t.Fatalf("reload from sqlite: %v", err)
	}

	var stored []byte
	if err := db.QueryRow(`SELECT wrapped_key FROM data_keys WHERE scope = 'env:7'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if string(stored) == string(first[:]) || len(stored) == 0 {
		t.Fatal("data key must be stored wrapped")
	}

	other, _ := crypto.NewKey()
	if err := secrets.NewKeyring(other, DataKeys{DB: db}, time.Now).Verify(ctx); err == nil {
		t.Fatal("verify must fail with another master key")
	}
}

func TestInsertDataKeyKeepsFirst(t *testing.T) {
	r := DataKeys{DB: migratedDB(t)}
	ctx := context.Background()
	a := secrets.WrappedKey{Scope: "env:1", Sealed: crypto.Sealed{Nonce: []byte("n1"), Ciphertext: []byte("c1")}, MasterKeyID: "m", CreatedAt: time.Now()}
	b := a
	b.Sealed = crypto.Sealed{Nonce: []byte("n2"), Ciphertext: []byte("c2")}
	if err := r.InsertDataKey(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := r.InsertDataKey(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetDataKey(ctx, "env:1")
	if err != nil || string(got.Sealed.Ciphertext) != "c1" {
		t.Fatalf("got %+v, %v", got, err)
	}
}
