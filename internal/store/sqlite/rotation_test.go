package sqlite

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

func TestRotationKeepsSecretsReadable(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	oldMaster, _ := crypto.NewKey()
	newMaster, _ := crypto.NewKey()
	storeWith := func(master crypto.Key) *secrets.Builtin {
		return &secrets.Builtin{Keys: secrets.NewKeyring(master, DataKeys{DB: db}, time.Now), Store: SecretVersions{DB: db}, Now: time.Now}
	}

	want := map[string]string{"DB_PASSWORD": "pw"}
	for _, s := range []secrets.Scope{secrets.EnvScope(1), secrets.AppScope(1)} {
		if _, err := storeWith(oldMaster).Update(ctx, s, secrets.Changes{Set: want}, nil); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := secrets.Rotate(ctx, DataKeys{DB: db}, oldMaster, newMaster); err != nil || n != 2 {
		t.Fatalf("rotate: %d %v", n, err)
	}

	for _, s := range []secrets.Scope{secrets.EnvScope(1), secrets.AppScope(1)} {
		got, err := storeWith(newMaster).Values(ctx, s, 1)
		if err != nil || !maps.Equal(got, want) {
			t.Fatalf("%s after rotation: %v %v", s, got, err)
		}
		if _, err := storeWith(oldMaster).Values(ctx, s, 1); err == nil {
			t.Fatalf("%s still readable with the old key", s)
		}
	}
}
