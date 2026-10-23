package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
)

// TokenService manages API tokens.
type TokenService interface {
	Create(ctx context.Context, u domain.User, name string) (string, domain.APIToken, error)
	List(ctx context.Context, userID int64) ([]domain.APIToken, error)
	Revoke(ctx context.Context, userID, id int64) (domain.APIToken, error)
	Authenticate(ctx context.Context, raw string) (auth.Principal, bool, error)
}

type tokenHandlers struct {
	logger *slog.Logger
	tokens TokenService
	audit  auditor
}

func (h tokenHandlers) register(rt *routes) {
	rt.handle("GET /tokens", h.list)
	rt.handle("POST /tokens", h.create)
	rt.handle("DELETE /tokens/{id}", h.revoke)
}

type tokenView struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
}

func toTokenView(t domain.APIToken) tokenView {
	return tokenView{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt, RevokedAt: t.RevokedAt}
}

func (h tokenHandlers) list(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	tokens, err := h.tokens.List(r.Context(), p.User.ID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]tokenView, 0, len(tokens))
	for _, t := range tokens {
		views = append(views, toTokenView(t))
	}
	writeJSON(w, http.StatusOK, map[string][]tokenView{"tokens": views})
}

type createTokenRequest struct {
	Name string `json:"name"`
}

type createTokenResponse struct {
	Token tokenView `json:"token"`
	// Secret is the raw bearer value. It is only ever returned here.
	Secret string `json:"secret"`
}

func (h tokenHandlers) create(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	raw, t, err := h.tokens.Create(r.Context(), p.User, req.Name)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, tokenEvent(domain.AuditTokenCreated, t))
	writeJSON(w, http.StatusCreated, createTokenResponse{Token: toTokenView(t), Secret: raw})
}

func (h tokenHandlers) revoke(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id: must be an integer")
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	t, err := h.tokens.Revoke(r.Context(), p.User.ID, id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, tokenEvent(domain.AuditTokenRevoked, t))
	w.WriteHeader(http.StatusNoContent)
}

func tokenEvent(action string, t domain.APIToken) domain.AuditEvent {
	return domain.AuditEvent{
		Action:     action,
		TargetType: "token",
		TargetID:   strconv.FormatInt(t.ID, 10),
		Metadata:   map[string]string{"name": t.Name},
	}
}
