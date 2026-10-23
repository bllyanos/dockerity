package api

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// fakeSessions accepts "pw" for user "alice" and tracks live raw tokens.
type fakeSessions struct {
	live map[string]domain.User
	next int
}

func newFakeSessions() *fakeSessions { return &fakeSessions{live: map[string]domain.User{}} }

var alice = domain.User{ID: 1, Username: "alice", IsAdmin: true, PasswordHash: "secret-hash"}

func (f *fakeSessions) Login(ctx context.Context, username, pw string) (string, domain.Session, domain.User, error) {
	if username != "alice" || pw != "pw" {
		return "", domain.Session{}, domain.User{}, domain.ErrUnauthorized
	}
	raw, sess, err := f.Start(ctx, alice)
	return raw, sess, alice, err
}

func (f *fakeSessions) Start(_ context.Context, u domain.User) (string, domain.Session, error) {
	f.next++
	raw := "dkts_" + string(rune('a'+f.next))
	f.live[raw] = u
	return raw, domain.Session{UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeSessions) Authenticate(_ context.Context, raw string) (auth.Principal, domain.Session, error) {
	u, ok := f.live[raw]
	if !ok {
		return auth.Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	return auth.Principal{User: u}, domain.Session{UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeSessions) Logout(_ context.Context, raw string) error {
	delete(f.live, raw)
	return nil
}

// fakeAudit keeps events in memory, newest last.
type fakeAudit struct{ events []domain.AuditEvent }

func (f *fakeAudit) Record(_ context.Context, e domain.AuditEvent) error {
	e.ID = int64(len(f.events) + 1)
	f.events = append(f.events, e)
	return nil
}

func (f *fakeAudit) ListAudit(_ context.Context, before int64, limit int) ([]domain.AuditEvent, error) {
	var out []domain.AuditEvent
	for i := len(f.events) - 1; i >= 0 && len(out) < limit; i-- {
		if before <= 0 || f.events[i].ID < before {
			out = append(out, f.events[i])
		}
	}
	return out, nil
}

func (f *fakeAudit) actions() []string {
	var out []string
	for _, e := range f.events {
		out = append(out, e.Action)
	}
	return out
}

// fakeTokens issues "dkta_<n>" tokens for any user.
type fakeTokens struct {
	tokens []domain.APIToken
	raw    map[string]int64
	users  map[int64]domain.User
	used   map[int64]bool
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{raw: map[string]int64{}, users: map[int64]domain.User{}, used: map[int64]bool{}}
}

func (f *fakeTokens) Create(_ context.Context, u domain.User, name string) (string, domain.APIToken, error) {
	if name == "" {
		return "", domain.APIToken{}, domain.Invalid("name", "required")
	}
	t := domain.APIToken{ID: int64(len(f.tokens) + 1), UserID: u.ID, Name: name, TokenDigest: "digest"}
	f.tokens = append(f.tokens, t)
	raw := "dkta_" + string(rune('a'+t.ID))
	f.raw[raw] = t.ID
	f.users[u.ID] = u
	return raw, t, nil
}

func (f *fakeTokens) List(_ context.Context, userID int64) ([]domain.APIToken, error) {
	var out []domain.APIToken
	for _, t := range f.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTokens) Revoke(_ context.Context, userID, id int64) (domain.APIToken, error) {
	for i, t := range f.tokens {
		if t.ID == id && t.UserID == userID && !t.Revoked() {
			now := time.Now()
			f.tokens[i].RevokedAt = &now
			return f.tokens[i], nil
		}
	}
	return domain.APIToken{}, domain.ErrNotFound
}

func (f *fakeTokens) Authenticate(_ context.Context, raw string) (auth.Principal, bool, error) {
	id, ok := f.raw[raw]
	if !ok || f.tokens[id-1].Revoked() {
		return auth.Principal{}, false, domain.ErrUnauthorized
	}
	first := !f.used[id]
	f.used[id] = true
	return auth.Principal{User: f.users[f.tokens[id-1].UserID], APITokenID: id}, first, nil
}

// fakeUserAdmin wraps a slice of users.
type fakeUserAdmin struct{ users []domain.User }

func (f *fakeUserAdmin) List(context.Context) ([]domain.User, error) { return f.users, nil }

func (f *fakeUserAdmin) Create(_ context.Context, username, _ string, isAdmin bool) (domain.User, error) {
	u := domain.User{ID: int64(len(f.users) + 1), Username: username, IsAdmin: isAdmin, PasswordHash: "secret-hash"}
	f.users = append(f.users, u)
	return u, nil
}

func (f *fakeUserAdmin) SetDisabled(_ context.Context, actor domain.User, id int64, disabled bool) (domain.User, error) {
	if actor.ID == id && disabled {
		return domain.User{}, domain.Invalid("id", "you cannot disable your own account")
	}
	if id < 1 || int(id) > len(f.users) {
		return domain.User{}, domain.ErrNotFound
	}
	if disabled {
		now := time.Now()
		f.users[id-1].DisabledAt = &now
	} else {
		f.users[id-1].DisabledAt = nil
	}
	return f.users[id-1], nil
}

func (f *fakeUserAdmin) ResetPassword(_ context.Context, id int64, _ string) (domain.User, error) {
	if id < 1 || int(id) > len(f.users) {
		return domain.User{}, domain.ErrNotFound
	}
	return f.users[id-1], nil
}

// fakeApps keeps apps in a slice and applies only the duplicate-name rule.
type fakeApps struct{ apps []domain.App }

func (f *fakeApps) List(context.Context) ([]domain.App, error) { return f.apps, nil }

func (f *fakeApps) Get(_ context.Context, id int64) (domain.App, error) {
	for _, a := range f.apps {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.App{}, domain.ErrNotFound
}

func (f *fakeApps) Create(_ context.Context, in apps.CreateInput) (domain.App, error) {
	for _, a := range f.apps {
		if a.Name == in.Name {
			return domain.App{}, domain.ErrConflict
		}
	}
	if in.Name == "" {
		return domain.App{}, domain.Invalid("name", "required")
	}
	a := domain.App{ID: int64(len(f.apps) + 1), Name: in.Name, RepoURL: in.RepoURL, DefaultBranch: in.DefaultBranch, DeployPublicKey: "ssh-ed25519 AAAA"}
	f.apps = append(f.apps, a)
	return a, nil
}

func (f *fakeApps) Update(ctx context.Context, id int64, in apps.UpdateInput) (domain.App, error) {
	a, err := f.Get(ctx, id)
	if err != nil {
		return a, err
	}
	if in.DefaultBranch != nil {
		a.DefaultBranch = *in.DefaultBranch
	}
	f.apps[id-1] = a
	return a, nil
}

func (f *fakeApps) Delete(ctx context.Context, id int64) (domain.App, error) {
	return f.Get(ctx, id)
}
