package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/domain"
)

type memUserLookup struct{ users []domain.User }

func (m *memUserLookup) GetUserByID(_ context.Context, id int64) (domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (m *memUserLookup) GetUserByUsername(_ context.Context, name string) (domain.User, error) {
	for _, u := range m.users {
		if u.Username == name {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

type memSessions struct {
	m       map[string]domain.Session
	touches int
}

func newMemSessions() *memSessions { return &memSessions{m: map[string]domain.Session{}} }

func (m *memSessions) CreateSession(_ context.Context, s domain.Session) error {
	m.m[s.TokenDigest] = s
	return nil
}

func (m *memSessions) GetSession(_ context.Context, d string) (domain.Session, error) {
	s, ok := m.m[d]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return s, nil
}

func (m *memSessions) TouchSession(_ context.Context, d string, seen, exp time.Time) error {
	s := m.m[d]
	s.LastSeenAt, s.ExpiresAt = seen, exp
	m.m[d] = s
	m.touches++
	return nil
}

func (m *memSessions) DeleteSession(_ context.Context, d string) error {
	delete(m.m, d)
	return nil
}

func (m *memSessions) DeleteExpiredSessions(_ context.Context, now time.Time) error {
	for d, s := range m.m {
		if s.Expired(now) {
			delete(m.m, d)
		}
	}
	return nil
}

type sessionFixture struct {
	svc   *Sessions
	users *memUserLookup
	store *memSessions
	now   time.Time
}

func newSessionFixture(t *testing.T) *sessionFixture {
	t.Helper()
	hash, err := password.Hash("correct password", fastParams)
	if err != nil {
		t.Fatal(err)
	}
	f := &sessionFixture{
		users: &memUserLookup{users: []domain.User{{ID: 1, Username: "alice", PasswordHash: hash}}},
		store: newMemSessions(),
		now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	f.svc = &Sessions{
		Users: f.users, Store: f.store, PasswordParams: fastParams, TTL: time.Hour,
		Now: func() time.Time { return f.now },
	}
	return f
}

func TestLoginAndAuthenticate(t *testing.T) {
	f := newSessionFixture(t)
	ctx := context.Background()

	raw, sess, u, err := f.svc.Login(ctx, " Alice ", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 || !sess.ExpiresAt.Equal(f.now.Add(time.Hour)) {
		t.Fatalf("unexpected session %+v for %+v", sess, u)
	}
	if _, ok := f.store.m[raw]; ok {
		t.Fatal("raw token must not be stored")
	}

	p, _, err := f.svc.Authenticate(ctx, raw)
	if err != nil || p.User.ID != 1 {
		t.Fatalf("authenticate: %+v, %v", p, err)
	}
}

func TestLoginFailures(t *testing.T) {
	f := newSessionFixture(t)
	ctx := context.Background()
	for name, creds := range map[string][2]string{
		"unknown user":   {"bob", "correct password"},
		"wrong password": {"alice", "wrong password!!"},
	} {
		if _, _, _, err := f.svc.Login(ctx, creds[0], creds[1]); !errors.Is(err, domain.ErrUnauthorized) {
			t.Errorf("%s: got %v", name, err)
		}
	}

	disabled := f.now
	f.users.users[0].DisabledAt = &disabled
	if _, _, _, err := f.svc.Login(ctx, "alice", "correct password"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("disabled user: got %v", err)
	}
}

func TestAuthenticateRejects(t *testing.T) {
	f := newSessionFixture(t)
	ctx := context.Background()
	raw, _, _, _ := f.svc.Login(ctx, "alice", "correct password")

	for _, bad := range []string{"", "garbage", "dkta_notasession", "dkts_unknown"} {
		if _, _, err := f.svc.Authenticate(ctx, bad); !errors.Is(err, domain.ErrUnauthorized) {
			t.Errorf("%q: got %v", bad, err)
		}
	}

	f.now = f.now.Add(time.Hour)
	if _, _, err := f.svc.Authenticate(ctx, raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expired: got %v", err)
	}
}

func TestAuthenticateRejectsDisabledUser(t *testing.T) {
	f := newSessionFixture(t)
	ctx := context.Background()
	raw, _, _, _ := f.svc.Login(ctx, "alice", "correct password")
	disabled := f.now
	f.users.users[0].DisabledAt = &disabled
	if _, _, err := f.svc.Authenticate(ctx, raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestAuthenticateSlidesExpiry(t *testing.T) {
	f := newSessionFixture(t)
	ctx := context.Background()
	raw, _, _, _ := f.svc.Login(ctx, "alice", "correct password")

	f.now = f.now.Add(10 * time.Second)
	if _, _, err := f.svc.Authenticate(ctx, raw); err != nil {
		t.Fatal(err)
	}
	if f.store.touches != 0 {
		t.Fatal("touched before touchInterval")
	}

	f.now = f.now.Add(50 * time.Minute)
	_, sess, err := f.svc.Authenticate(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	if f.store.touches != 1 || !sess.ExpiresAt.Equal(f.now.Add(time.Hour)) {
		t.Fatalf("expiry not extended: %+v (touches %d)", sess, f.store.touches)
	}

	f.now = f.now.Add(59 * time.Minute)
	if _, _, err := f.svc.Authenticate(ctx, raw); err != nil {
		t.Fatalf("session should still be valid after sliding: %v", err)
	}
}

func TestLogout(t *testing.T) {
	f := newSessionFixture(t)
	ctx := context.Background()
	raw, _, _, _ := f.svc.Login(ctx, "alice", "correct password")
	if err := f.svc.Logout(ctx, raw); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.Authenticate(ctx, raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("got %v after logout", err)
	}
	if err := f.svc.Logout(ctx, "garbage"); err != nil {
		t.Fatalf("logout of unknown token: %v", err)
	}
}
