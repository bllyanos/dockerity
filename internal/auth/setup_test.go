package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/domain"
)

var fastParams = password.Params{Memory: 1024, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}

var fixedNow = func() time.Time { return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) }

type memUsers struct{ users []domain.User }

func (m *memUsers) CountUsers(context.Context) (int, error) { return len(m.users), nil }

func (m *memUsers) CreateFirstAdmin(_ context.Context, u domain.User) (domain.User, error) {
	if len(m.users) > 0 {
		return domain.User{}, domain.ErrConflict
	}
	u.ID = int64(len(m.users) + 1)
	m.users = append(m.users, u)
	return u, nil
}

func TestSetupRun(t *testing.T) {
	store := &memUsers{}
	s := &Setup{Users: store, PasswordParams: fastParams, Now: fixedNow}
	ctx := context.Background()

	if needed, _ := s.Needed(ctx); !needed {
		t.Fatal("setup should be needed on an empty store")
	}
	u, err := s.Run(ctx, "  Admin ", "a long enough password")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "admin" || !u.IsAdmin || !u.CreatedAt.Equal(fixedNow()) {
		t.Fatalf("unexpected user %+v", u)
	}
	if err := password.Verify("a long enough password", u.PasswordHash); err != nil {
		t.Fatalf("stored hash does not verify: %v", err)
	}
	if needed, _ := s.Needed(ctx); needed {
		t.Fatal("setup should not be needed after it ran")
	}
	if _, err := s.Run(ctx, "other", "a long enough password"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second run: got %v, want ErrConflict", err)
	}
}

func TestSetupValidation(t *testing.T) {
	s := &Setup{Users: &memUsers{}, PasswordParams: fastParams, Now: fixedNow}
	cases := map[string][2]string{
		"username": {"x", "a long enough password"},
		"password": {"admin", "short"},
	}
	for field, in := range cases {
		_, err := s.Run(context.Background(), in[0], in[1])
		var ve *domain.ValidationError
		if !errors.As(err, &ve) || ve.Field != field {
			t.Errorf("%s: got %v", field, err)
		}
	}
}
