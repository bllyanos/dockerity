package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/auth/ratelimit"
	"github.com/bllyanos/dockerity/internal/domain"
)

type authHandlers struct {
	logger   *slog.Logger
	sessions SessionService
	limiter  *ratelimit.Limiter
}

func (h authHandlers) register(rt *routes) {
	rt.handlePublic("POST /auth/login", h.login)
	rt.handle("POST /auth/logout", h.logout)
	rt.handle("GET /auth/me", h.me)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userKey := "user:" + domain.NormalizeUsername(req.Username)
	ipKey := "ip:" + clientIP(r)
	if !h.limiter.Allowed(userKey) || !h.limiter.Allowed(ipKey) {
		writeError(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
		return
	}

	raw, sess, u, err := h.sessions.Login(r.Context(), req.Username, req.Password)
	if errors.Is(err, domain.ErrUnauthorized) {
		h.limiter.Fail(userKey)
		h.limiter.Fail(ipKey)
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.limiter.Reset(userKey)
	setSessionCookie(w, r, raw, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]userView{"user": toUserView(u)})
}

func (h authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Logout(r.Context(), sessionToken(r)); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandlers) me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	writeJSON(w, http.StatusOK, map[string]userView{"user": toUserView(p.User)})
}
