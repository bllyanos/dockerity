package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type memTokens struct{ tokens []domain.APIToken }

func (m *memTokens) CreateAPIToken(_ context.Context, t domain.APIToken) (domain.APIToken, error) {
	t.ID = int64(len(m.tokens) + 1)
	m.tokens = append(m.tokens, t)
	return t, nil
}

func (m *memTokens) ListAPITokens(_ context.Context, userID int64) ([]domain.APIToken, error) {
	var out []domain.APIToken
	for _, t := range m.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memTokens) GetAPITokenByDigest(_ context.Context, d string) (domain.APIToken, error) {
	for _, t := range m.tokens {
		if t.TokenDigest == d {
			return t, nil
		}
	}
	return domain.APIToken{}, domain.ErrNotFound
}

func (m *memTokens) RevokeAPIToken(_ context.Context, userID, id int64, at time.Time) (domain.APIToken, error) {
	for i, t := range m.tokens {
		if t.ID == id && t.UserID == userID && !t.Revoked() {
			m.tokens[i].RevokedAt = &at
			return m.tokens[i], nil
		}
	}
	return domain.APIToken{}, domain.ErrNotFound
}

func (m *memTokens) TouchAPIToken(_ context.Context, id int64, at time.Time) error {
	m.tokens[id-1].LastUsedAt = &at
	return nil
}

func newTokenFixture() (*Tokens, *memUserLookup, *time.Time) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	users := &memUserLookup{users: []domain.User{{ID: 1, Username: "alice"}}}
	return &Tokens{Users: users, Store: &memTokens{}, Now: func() time.Time { return now }}, users, &now
}

func TestTokenCreateAndAuthenticate(t *testing.T) {
	s, _, now := newTokenFixture()
	ctx := context.Background()

	raw, tok, err := s.Create(ctx, domain.User{ID: 1}, "  ci  ")
	if err != nil {
		t.Fatal(err)
	}
	if tok.Name != "ci" || tok.TokenDigest == raw {
		t.Fatalf("unexpected token %+v", tok)
	}

	p, first, err := s.Authenticate(ctx, raw)
	if err != nil || p.User.ID != 1 || p.APITokenID != tok.ID || !first {
		t.Fatalf("first use: %+v %v %v", p, first, err)
	}
	*now = now.Add(30 * time.Second)
	if _, first, _ = s.Authenticate(ctx, raw); first {
		t.Fatal("use within interval must not count as first use")
	}
	*now = now.Add(time.Minute)
	if _, first, _ = s.Authenticate(ctx, raw); !first {
		t.Fatal("use after interval should be recorded again")
	}
}

func TestTokenRejected(t *testing.T) {
	s, users, now := newTokenFixture()
	ctx := context.Background()
	raw, tok, _ := s.Create(ctx, domain.User{ID: 1}, "ci")

	for _, bad := range []string{"", "dkts_session", "dkta_unknown"} {
		if _, _, err := s.Authenticate(ctx, bad); !errors.Is(err, domain.ErrUnauthorized) {
			t.Errorf("%q: got %v", bad, err)
		}
	}

	users.users[0].DisabledAt = now
	if _, _, err := s.Authenticate(ctx, raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("disabled user: got %v", err)
	}
	users.users[0].DisabledAt = nil

	if _, err := s.Revoke(ctx, 2, tok.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revoke by other user: got %v", err)
	}
	if _, err := s.Revoke(ctx, 1, tok.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Authenticate(ctx, raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("revoked: got %v", err)
	}
}

func TestTokenNameValidated(t *testing.T) {
	s, _, _ := newTokenFixture()
	var ve *domain.ValidationError
	if _, _, err := s.Create(context.Background(), domain.User{ID: 1}, "   "); !errors.As(err, &ve) {
		t.Fatalf("got %v", err)
	}
}
