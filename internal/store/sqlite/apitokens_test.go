package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestAPITokensLifecycle(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	u, _ := Users{DB: db}.CreateFirstAdmin(ctx, newUser("alice"))
	r := APITokens{DB: db}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	a, err := r.CreateAPIToken(ctx, domain.APIToken{UserID: u.ID, Name: "ci", TokenDigest: "da", CreatedAt: t0})
	if err != nil || a.ID == 0 {
		t.Fatalf("create: %+v %v", a, err)
	}
	b, _ := r.CreateAPIToken(ctx, domain.APIToken{UserID: u.ID, Name: "laptop", TokenDigest: "db", CreatedAt: t0})

	list, err := r.ListAPITokens(ctx, u.ID)
	if err != nil || len(list) != 2 || list[0].ID != b.ID {
		t.Fatalf("list: %+v %v", list, err)
	}

	if err := r.TouchAPIToken(ctx, a.ID, t0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetAPITokenByDigest(ctx, "da")
	if err != nil || got.LastUsedAt == nil || !got.LastUsedAt.Equal(t0.Add(time.Hour)) {
		t.Fatalf("get after touch: %+v %v", got, err)
	}

	revoked, err := r.RevokeAPIToken(ctx, u.ID, a.ID, t0.Add(2*time.Hour))
	if err != nil || !revoked.Revoked() {
		t.Fatalf("revoke: %+v %v", revoked, err)
	}
	if _, err := r.RevokeAPIToken(ctx, u.ID, a.ID, t0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second revoke: %v", err)
	}
	if _, err := r.RevokeAPIToken(ctx, u.ID+1, b.ID, t0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("revoke by other user: %v", err)
	}

	if err := r.RevokeUserAPITokens(ctx, u.ID, t0); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.GetAPITokenByDigest(ctx, "db"); !got.Revoked() {
		t.Fatal("RevokeUserAPITokens left a token active")
	}
	if _, err := r.GetAPITokenByDigest(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}
