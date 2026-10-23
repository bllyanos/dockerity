package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func newUser(name string) domain.User {
	return domain.User{
		Username:     name,
		PasswordHash: "$argon2id$hash",
		IsAdmin:      true,
		CreatedAt:    time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC),
	}
}

func TestCreateFirstAdminOnlyOnce(t *testing.T) {
	r := Users{DB: migratedDB(t)}
	ctx := context.Background()

	u, err := r.CreateFirstAdmin(ctx, newUser("alice"))
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 {
		t.Fatal("ID not set")
	}
	if _, err := r.CreateFirstAdmin(ctx, newUser("bob")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("got %v, want ErrConflict", err)
	}
	if n, _ := r.CountUsers(ctx); n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
}

func TestCreateFirstAdminConcurrent(t *testing.T) {
	r := Users{DB: migratedDB(t)}
	ctx := context.Background()

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() {
			_, errs[i] = r.CreateFirstAdmin(ctx, newUser("user"+string(rune('a'+i))+"x"))
		})
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, domain.ErrConflict):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d setups succeeded, want exactly 1", ok)
	}
}

func TestGetUser(t *testing.T) {
	r := Users{DB: migratedDB(t)}
	ctx := context.Background()
	created, _ := r.CreateFirstAdmin(ctx, newUser("alice"))

	byName, err := r.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	byID, err := r.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range []domain.User{byName, byID} {
		if got.ID != created.ID || got.Username != "alice" || !got.IsAdmin || got.Disabled() ||
			!got.CreatedAt.Equal(created.CreatedAt) {
			t.Fatalf("got %+v, want %+v", got, created)
		}
	}
	if _, err := r.GetUserByUsername(ctx, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
