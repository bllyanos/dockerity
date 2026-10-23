package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestUserAdminQueries(t *testing.T) {
	r := Users{DB: migratedDB(t)}
	ctx := context.Background()

	admin, _ := r.CreateFirstAdmin(ctx, newUser("zed"))
	bob, err := r.CreateUser(ctx, domain.User{Username: "bob", PasswordHash: "h", CreatedAt: time.Now()})
	if err != nil || bob.ID == 0 {
		t.Fatalf("create: %+v %v", bob, err)
	}
	if _, err := r.CreateUser(ctx, domain.User{Username: "bob", PasswordHash: "h", CreatedAt: time.Now()}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}

	list, err := r.ListUsers(ctx)
	if err != nil || len(list) != 2 || list[0].Username != "bob" || list[1].ID != admin.ID {
		t.Fatalf("list: %+v %v", list, err)
	}

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := r.SetUserDisabled(ctx, bob.ID, &at); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.GetUserByID(ctx, bob.ID); !got.Disabled() {
		t.Fatal("not disabled")
	}
	if err := r.SetUserDisabled(ctx, bob.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.GetUserByID(ctx, bob.ID); got.Disabled() {
		t.Fatal("not enabled")
	}

	if err := r.SetUserPassword(ctx, bob.ID, "new-hash"); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.GetUserByID(ctx, bob.ID); got.PasswordHash != "new-hash" {
		t.Fatal("password not updated")
	}
	if err := r.SetUserPassword(ctx, 999, "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}
}
