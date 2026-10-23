package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
)

// SessionService manages password logins and cookie sessions.
type SessionService interface {
	Login(ctx context.Context, username, password string) (string, domain.Session, domain.User, error)
	Start(ctx context.Context, u domain.User) (string, domain.Session, error)
	Authenticate(ctx context.Context, raw string) (auth.Principal, domain.Session, error)
	Logout(ctx context.Context, raw string) error
}

// authGuard lets public routes through and requires a valid API token or
// session for everything else. Missing services deny by default.
type authGuard struct {
	logger   *slog.Logger
	routes   *routes
	sessions SessionService
	tokens   TokenService
	audit    auditor
}

func (g authGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if g.routes.isPublic(r) {
		g.routes.mux.ServeHTTP(w, r)
		return
	}
	if raw, ok := bearerToken(r); ok {
		g.serveWithToken(w, r, raw)
		return
	}
	g.serveWithSession(w, r)
}

func (g authGuard) serveWithToken(w http.ResponseWriter, r *http.Request, raw string) {
	if g.tokens == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	p, firstUse, err := g.tokens.Authenticate(r.Context(), raw)
	if errors.Is(err, domain.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeServiceError(w, r, g.logger, err)
		return
	}
	r = r.WithContext(auth.WithPrincipal(r.Context(), p))
	if firstUse {
		g.audit.record(r, domain.AuditEvent{
			Action:     domain.AuditTokenUsed,
			TargetType: "token",
			TargetID:   strconv.FormatInt(p.APITokenID, 10),
		})
	}
	g.routes.mux.ServeHTTP(w, r)
}

func (g authGuard) serveWithSession(w http.ResponseWriter, r *http.Request) {
	raw := sessionToken(r)
	if g.sessions == nil || raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	p, sess, err := g.sessions.Authenticate(r.Context(), raw)
	if errors.Is(err, domain.ErrUnauthorized) {
		clearSessionCookie(w, r)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeServiceError(w, r, g.logger, err)
		return
	}
	// Keep the browser cookie in step with the sliding server-side expiry.
	setSessionCookie(w, r, raw, sess.ExpiresAt)
	g.routes.mux.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
}

// bearerToken returns the token from "Authorization: Bearer <token>".
// ok is true whenever the header uses the Bearer scheme, even if the value is
// empty, so a bad token never falls back to the session cookie.
func bearerToken(r *http.Request) (string, bool) {
	scheme, value, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	return strings.TrimSpace(value), true
}
