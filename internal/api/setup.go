package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/domain"
)

// SetupService runs the first-run setup.
type SetupService interface {
	Needed(ctx context.Context) (bool, error)
	Run(ctx context.Context, username, password string) (domain.User, error)
}

type setupHandlers struct {
	logger *slog.Logger
	setup  SetupService
	// sessions, when set, logs the new admin in right after setup.
	sessions SessionService
}

func (h setupHandlers) register(rt *routes) {
	rt.handlePublic("GET /setup", h.status)
	rt.handlePublic("POST /setup", h.run)
}

func (h setupHandlers) status(w http.ResponseWriter, r *http.Request) {
	needed, err := h.setup.Needed(r.Context())
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"needed": needed})
}

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h setupHandlers) run(w http.ResponseWriter, r *http.Request) {
	var req setupRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := h.setup.Run(r.Context(), req.Username, req.Password)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	if h.sessions != nil {
		raw, sess, err := h.sessions.Start(r.Context(), u)
		if err != nil {
			writeServiceError(w, r, h.logger, err)
			return
		}
		setSessionCookie(w, r, raw, sess.ExpiresAt)
	}
	writeJSON(w, http.StatusCreated, map[string]userView{"user": toUserView(u)})
}
