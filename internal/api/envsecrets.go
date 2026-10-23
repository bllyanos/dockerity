package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/envsecrets"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// EnvSecretService manages environment secrets. Values are write-only.
type EnvSecretService interface {
	Current(ctx context.Context, envID int64) (envsecrets.Summary, error)
	Update(ctx context.Context, envID int64, changes secrets.Changes, actor *int64) (envsecrets.Summary, error)
	Versions(ctx context.Context, envID int64) ([]secrets.Version, error)
}

type envSecretHandlers struct {
	logger  *slog.Logger
	secrets EnvSecretService
	audit   auditor
}

func (h envSecretHandlers) register(rt *routes) {
	rt.handle("GET /environments/{id}/secrets", h.current)
	rt.handle("PATCH /environments/{id}/secrets", h.update)
	rt.handle("GET /environments/{id}/secrets/versions", h.versions)
}

// secretsView never carries values, only names.
type secretsView struct {
	Version int      `json:"version"`
	Keys    []string `json:"keys"`
}

type secretVersionView struct {
	Version   int       `json:"version"`
	Keys      []string  `json:"keys"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy *int64    `json:"createdBy"`
}

func (h envSecretHandlers) current(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s, err := h.secrets.Current(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]secretsView{"secrets": {Version: s.Version, Keys: s.Keys}})
}

type updateSecretsRequest struct {
	Set   map[string]string `json:"set"`
	Unset []string          `json:"unset"`
}

func (h envSecretHandlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req updateSecretsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	changes := secrets.Changes{Set: req.Set, Unset: req.Unset}
	p, _ := auth.PrincipalFrom(r.Context())
	actor := p.User.ID
	s, err := h.secrets.Update(r.Context(), id, changes, &actor)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	meta := changes.AuditMetadata()
	meta["version"] = strconv.Itoa(s.Version)
	h.audit.record(r, domain.AuditEvent{
		Action: domain.AuditSecretsUpdated, TargetType: "environment", TargetID: strconv.FormatInt(id, 10), Metadata: meta,
	})
	writeJSON(w, http.StatusOK, map[string]secretsView{"secrets": {Version: s.Version, Keys: s.Keys}})
}

func (h envSecretHandlers) versions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	vs, err := h.secrets.Versions(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]secretVersionView, 0, len(vs))
	for _, v := range vs {
		views = append(views, secretVersionView{Version: v.Version, Keys: v.Keys, CreatedAt: v.CreatedAt, CreatedBy: v.CreatedBy})
	}
	writeJSON(w, http.StatusOK, map[string][]secretVersionView{"versions": views})
}
