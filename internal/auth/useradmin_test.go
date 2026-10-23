package auth

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/domain"
)

type memAdminUsers struct{ users []domain.User }

func (m *memAdminUsers) ListUsers(context.Context) ([]domain.User, error) { return m.users, nil }

func (m *memAdminUsers) GetUserByID(_ context.Context, id int64) (domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (m *memAdminUsers) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	for _, existing := range m.users {
		if existing.Username == u.Username {
			return domain.User{}, domain.ErrConflict
		}
	}
	u.ID = int64(len(m.users) + 1)
	m.users = append(m.users, u)
	return u, nil
}

func (m *memAdminUsers) SetUserDisabled(_ context.Context, id int64, at *time.Time) error {
	m.users[id-1].DisabledAt = at
	return nil
}

func (m *memAdminUsers) SetUserPassword(_ context.Context, id int64, hash string) error {
	m.users[id-1].PasswordHash = hash
	return nil
}

type recordingRevoker struct{ calls []string }

func (r *recordingRevoker) DeleteUserSessions(context.Context, int64) error {
	r.calls = append(r.calls, "sessions")
	return nil
}

func (r *recordingRevoker) RevokeUserAPITokens(context.Context, int64, time.Time) error {
	r.calls = append(r.calls, "tokens")
	return nil
}

func newUserAdmin() (*UserAdmin, *memAdminUsers, *recordingRevoker) {
	users := &memAdminUsers{users: []domain.User{{ID: 1, Username: "admin", IsAdmin: true}}}
	rev := &recordingRevoker{}
	return &UserAdmin{Users: users, Credentials: rev, PasswordParams: fastParams, Now: fixedNow}, users, rev
}

func TestUserAdminCreate(t *testing.T) {
	s, _, _ := newUserAdmin()
	ctx := context.Background()
	u, err := s.Create(ctx, " Bob ", "a long enough password", false)
	if err != nil || u.Username != "bob" || u.IsAdmin {
		t.Fatalf("create: %+v %v", u, err)
	}
	if err := password.Verify("a long enough password", u.PasswordHash); err != nil {
		t.Fatal("hash does not verify")
	}
	if _, err := s.Create(ctx, "bob", "a long enough password", false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	var ve *domain.ValidationError
	if _, err := s.Create(ctx, "carol", "short", false); !errors.As(err, &ve) || ve.Field != "password" {
		t.Fatalf("weak password: %v", err)
	}
}

func TestUserAdminDisable(t *testing.T) {
	s, users, rev := newUserAdmin()
	ctx := context.Background()
	admin := users.users[0]
	bob, _ := s.Create(ctx, "bob", "a long enough password", false)

	var ve *domain.ValidationError
	if _, err := s.SetDisabled(ctx, admin, admin.ID, true); !errors.As(err, &ve) {
		t.Fatalf("self disable: %v", err)
	}

	got, err := s.SetDisabled(ctx, admin, bob.ID, true)
	if err != nil || !got.Disabled() {
		t.Fatalf("disable: %+v %v", got, err)
	}
	if !slices.Equal(rev.calls, []string{"sessions", "tokens"}) {
		t.Fatalf("credentials not revoked: %v", rev.calls)
	}

	got, err = s.SetDisabled(ctx, admin, bob.ID, false)
	if err != nil || got.Disabled() {
		t.Fatalf("enable: %+v %v", got, err)
	}
	if _, err := s.SetDisabled(ctx, admin, 99, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestUserAdminResetPassword(t *testing.T) {
	s, users, rev := newUserAdmin()
	ctx := context.Background()
	if _, err := s.ResetPassword(ctx, 1, "a brand new password"); err != nil {
		t.Fatal(err)
	}
	if err := password.Verify("a brand new password", users.users[0].PasswordHash); err != nil {
		t.Fatal("new password does not verify")
	}
	if !slices.Equal(rev.calls, []string{"sessions"}) {
		t.Fatalf("expected sessions to be ended only: %v", rev.calls)
	}
}
