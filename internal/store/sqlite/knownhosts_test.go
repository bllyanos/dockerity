package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/gitx"
)

func TestKnownHosts(t *testing.T) {
	r := KnownHosts{DB: migratedDB(t)}
	ctx := context.Background()
	if _, err := r.GetHostKey(ctx, "github.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty: %v", err)
	}
	k := gitx.KnownHost{Host: "github.com", KeyType: "ssh-ed25519", Key: "AAAA", Fingerprint: "SHA256:one", AddedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := r.AddHostKey(ctx, k); err != nil {
		t.Fatal(err)
	}
	other := k
	other.Fingerprint = "SHA256:two"
	if err := r.AddHostKey(ctx, other); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetHostKey(ctx, "github.com")
	if err != nil || got != k {
		t.Fatalf("first pin must win: %+v %v", got, err)
	}
}
