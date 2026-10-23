package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestSessionsLifecycle(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	u, err := Users{DB: db}.CreateFirstAdmin(ctx, newUser("alice"))
	if err != nil {
		t.Fatal(err)
	}
	r := Sessions{DB: db}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := domain.Session{TokenDigest: "d1", UserID: u.ID, CreatedAt: t0, LastSeenAt: t0, ExpiresAt: t0.Add(time.Hour)}

	if err := r.CreateSession(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetSession(ctx, "d1")
	if err != nil || got != s {
		t.Fatalf("got %+v, %v; want %+v", got, err, s)
	}

	t1 := t0.Add(30 * time.Minute)
	if err := r.TouchSession(ctx, "d1", t1, t1.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetSession(ctx, "d1")
	if !got.LastSeenAt.Equal(t1) || !got.ExpiresAt.Equal(t1.Add(time.Hour)) {
		t.Fatalf("touch not applied: %+v", got)
	}

	if err := r.DeleteSession(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetSession(ctx, "d1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestDeleteExpiredAndUserSessions(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	u, _ := Users{DB: db}.CreateFirstAdmin(ctx, newUser("alice"))
	r := Sessions{DB: db}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, exp := range []time.Duration{-time.Minute, time.Hour, 2 * time.Hour} {
		s := domain.Session{TokenDigest: string(rune('a' + i)), UserID: u.ID, CreatedAt: t0, LastSeenAt: t0, ExpiresAt: t0.Add(exp)}
		if err := r.CreateSession(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	if err := r.DeleteExpiredSessions(ctx, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetSession(ctx, "a"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("expired session not deleted")
	}
	if _, err := r.GetSession(ctx, "b"); err != nil {
		t.Fatal("live session deleted")
	}

	if err := r.DeleteUserSessions(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d sessions left", n)
	}
}
