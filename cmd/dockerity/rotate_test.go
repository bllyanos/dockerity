package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/config"
	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
)

func writeKey(t *testing.T, path string) crypto.Key {
	t.Helper()
	k, _ := crypto.NewKey()
	if err := os.WriteFile(path, []byte(k.Encode()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return k
}

func TestRotateMasterKeyCommand(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	oldKey := writeKey(t, filepath.Join(dir, "old"))
	newKey := writeKey(t, filepath.Join(dir, "new"))
	t.Setenv(config.EnvDataDir, dir)
	t.Setenv(config.EnvMasterKeyFile, filepath.Join(dir, "old"))

	db, err := sqlite.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	ms, _ := sqlite.Migrations()
	if _, err := sqlite.Migrate(ctx, db, ms); err != nil {
		t.Fatal(err)
	}
	store := &secrets.Builtin{Keys: secrets.NewKeyring(oldKey, sqlite.DataKeys{DB: db}, time.Now), Store: sqlite.SecretVersions{DB: db}, Now: time.Now}
	if _, err := store.Update(ctx, secrets.EnvScope(1), secrets.Changes{Set: map[string]string{"TOKEN": "t"}}, nil); err != nil {
		t.Fatal(err)
	}
	db.Close()

	var out bytes.Buffer
	if err := rotateMasterKey(ctx, []string{"--new-key-file", filepath.Join(dir, "new")}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Rotated 1 data key(s) to master key "+newKey.ID()) {
		t.Fatalf("unexpected output: %s", out.String())
	}

	db, _ = sqlite.Open(ctx, dir)
	defer db.Close()
	ring := secrets.NewKeyring(newKey, sqlite.DataKeys{DB: db}, time.Now)
	if err := ring.Verify(ctx); err != nil {
		t.Fatalf("new key rejected: %v", err)
	}
	got, err := (&secrets.Builtin{Keys: ring, Store: sqlite.SecretVersions{DB: db}, Now: time.Now}).Values(ctx, secrets.EnvScope(1), 1)
	if err != nil || got["TOKEN"] != "t" {
		t.Fatalf("secrets unreadable after rotation: %v %v", got, err)
	}
}
