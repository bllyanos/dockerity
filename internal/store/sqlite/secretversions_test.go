package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

func newSecretStore(t *testing.T, db *sql.DB) *secrets.Builtin {
	t.Helper()
	master, _ := crypto.NewKey()
	return &secrets.Builtin{
		Keys:  secrets.NewKeyring(master, DataKeys{DB: db}, time.Now),
		Store: SecretVersions{DB: db},
		Now:   time.Now,
	}
}

func TestSecretVersionsAreImmutable(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	store := newSecretStore(t, db)
	scope := secrets.EnvScope(1)

	if _, err := store.Latest(ctx, scope); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty scope: %v", err)
	}
	v1, err := store.Update(ctx, scope, secrets.Changes{Set: map[string]string{"DB_PASSWORD": "pw-one", "API_KEY": "k1"}}, nil)
	if err != nil || v1.Version != 1 {
		t.Fatalf("v1: %+v %v", v1, err)
	}
	v2, err := store.Update(ctx, scope, secrets.Changes{Set: map[string]string{"DB_PASSWORD": "pw-two"}, Unset: []string{"API_KEY"}}, nil)
	if err != nil || v2.Version != 2 {
		t.Fatalf("v2: %+v %v", v2, err)
	}

	old, err := store.Values(ctx, scope, 1)
	if err != nil || !maps.Equal(old, map[string]string{"DB_PASSWORD": "pw-one", "API_KEY": "k1"}) {
		t.Fatalf("v1 values: %v %v", old, err)
	}
	cur, _ := store.Values(ctx, scope, 2)
	if !maps.Equal(cur, map[string]string{"DB_PASSWORD": "pw-two"}) {
		t.Fatalf("v2 values: %v", cur)
	}

	versions, err := store.Versions(ctx, scope)
	if err != nil || len(versions) != 2 || versions[0].Version != 2 || len(versions[1].Keys) != 2 {
		t.Fatalf("versions: %+v %v", versions, err)
	}
	if latest, _ := store.Latest(ctx, scope); latest.Version != 2 {
		t.Fatalf("latest: %+v", latest)
	}
}

func TestSecretVersionConflict(t *testing.T) {
	r := SecretVersions{DB: migratedDB(t)}
	ctx := context.Background()
	v := secrets.EncryptedVersion{
		Version: secrets.Version{Scope: "env:1", Version: 1, Keys: []string{"A"}, CreatedAt: time.Now()},
		Sealed:  crypto.Sealed{Nonce: []byte("n"), Ciphertext: []byte("c")},
	}
	if err := r.InsertSecretVersion(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := r.InsertSecretVersion(ctx, v); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("got %v, want ErrConflict", err)
	}
}

// TestDatabaseHasNoPlaintextSecrets scans the raw SQLite files, including the WAL.
func TestDatabaseHasNoPlaintextSecrets(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ms, _ := Migrations()
	if _, err := Migrate(context.Background(), db, ms); err != nil {
		t.Fatal(err)
	}
	store := newSecretStore(t, db)
	const marker = "plaintext-marker-7f3a9c"
	for i := range 3 {
		ch := secrets.Changes{Set: map[string]string{"SECRET_VALUE": marker, "OTHER": marker + string(rune('a'+i))}}
		if _, err := store.Update(context.Background(), secrets.EnvScope(int64(i)), ch, nil); err != nil {
			t.Fatal(err)
		}
	}

	files, _ := filepath.Glob(filepath.Join(dir, FileName+"*"))
	if len(files) == 0 {
		t.Fatal("no database files found")
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(marker)) {
			t.Fatalf("plaintext secret found in %s", filepath.Base(f))
		}
	}
	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	if bytes.Contains(data, []byte(marker)) {
		t.Fatal("plaintext secret found after checkpoint")
	}
	if !bytes.Contains(data, []byte("SECRET_VALUE")) {
		t.Fatal("sanity check: key names are expected in clear")
	}
}
