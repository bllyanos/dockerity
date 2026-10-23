package api

import (
	"context"
	"io"
	"log/slog"
	"time"

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
