package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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

// requireAuth lets public routes through and requires a valid session for
// everything else. Without a SessionService every protected route is denied.
func requireAuth(logger *slog.Logger, rt *routes, sessions SessionService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rt.isPublic(r) {
			rt.mux.ServeHTTP(w, r)
			return
		}
		raw := sessionToken(r)
		if sessions == nil || raw == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		p, sess, err := sessions.Authenticate(r.Context(), raw)
		if errors.Is(err, domain.ErrUnauthorized) {
			clearSessionCookie(w, r)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err != nil {
			writeServiceError(w, r, logger, err)
			return
		}
		// Keep the browser cookie in step with the sliding server-side expiry.
		setSessionCookie(w, r, raw, sess.ExpiresAt)
		rt.mux.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
	})
}
