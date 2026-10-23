package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// EnvironmentService manages environments.
type EnvironmentService interface {
	List(ctx context.Context, appID int64) ([]domain.Environment, error)
	Get(ctx context.Context, id int64) (domain.Environment, error)
	CreatePermanent(ctx context.Context, appID int64, name, host string) (domain.Environment, error)
	UpdateDomain(ctx context.Context, id int64, host string) (domain.Environment, error)
	Delete(ctx context.Context, id int64, skipBackup bool) (domain.Environment, error)
	Extend(ctx context.Context, id int64) (domain.Environment, error)
}

type environmentHandlers struct {
	logger *slog.Logger
	envs   EnvironmentService
	audit  auditor
}

func (h environmentHandlers) register(rt *routes) {
	rt.handle("GET /apps/{id}/environments", h.list)
	rt.handle("POST /apps/{id}/environments", h.create)
	rt.handle("GET /environments/{id}", h.get)
	rt.handle("PATCH /environments/{id}", h.update)
	rt.handle("DELETE /environments/{id}", h.delete)
	rt.handle("POST /environments/{id}/extend", h.extend)
}

type environmentView struct {
	ID         int64          `json:"id"`
	AppID      int64          `json:"appId"`
	AppName    string         `json:"appName"`
	Name       string         `json:"name"`
	Type       domain.EnvType `json:"type"`
	Domain     string         `json:"domain"`
	TTLSeconds int64          `json:"ttlSeconds"`
	ExpiresAt  *time.Time     `json:"expiresAt"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

func toEnvView(e domain.Environment) environmentView {
	return environmentView{ID: e.ID, AppID: e.AppID, AppName: e.AppName, Name: e.Name, Type: e.Type,
		Domain: e.Domain, TTLSeconds: int64(e.TTL / time.Second), ExpiresAt: e.ExpiresAt, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt}
}

func envEvent(action string, e domain.Environment) domain.AuditEvent {
	return domain.AuditEvent{Action: action, TargetType: "environment", TargetID: strconv.FormatInt(e.ID, 10),
		Metadata: map[string]string{"app_id": strconv.FormatInt(e.AppID, 10), "name": e.Name, "domain": e.Domain}}
}

func (h environmentHandlers) list(w http.ResponseWriter, r *http.Request) {
	appID, ok := pathID(w, r)
	if !ok {
		return
	}
	envs, err := h.envs.List(r.Context(), appID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]environmentView, 0, len(envs))
	for _, e := range envs {
		views = append(views, toEnvView(e))
	}
	writeJSON(w, http.StatusOK, map[string][]environmentView{"environments": views})
}

type createEnvironmentRequest struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

func (h environmentHandlers) create(w http.ResponseWriter, r *http.Request) {
	appID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req createEnvironmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e, err := h.envs.CreatePermanent(r.Context(), appID, req.Name, req.Domain)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, envEvent(domain.AuditEnvironmentCreated, e))
	writeJSON(w, http.StatusCreated, map[string]environmentView{"environment": toEnvView(e)})
}

func (h environmentHandlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	e, err := h.envs.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]environmentView{"environment": toEnvView(e)})
}

type updateEnvironmentRequest struct {
	Domain string `json:"domain"`
}

func (h environmentHandlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req updateEnvironmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e, err := h.envs.UpdateDomain(r.Context(), id, req.Domain)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, envEvent(domain.AuditEnvironmentUpdated, e))
	writeJSON(w, http.StatusOK, map[string]environmentView{"environment": toEnvView(e)})
}

func (h environmentHandlers) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	skipBackup := r.URL.Query().Get("skipBackup") == "true"
	e, err := h.envs.Delete(r.Context(), id, skipBackup)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	ev := envEvent(domain.AuditEnvironmentDeleted, e)
	ev.Metadata["skip_backup"] = strconv.FormatBool(skipBackup)
	h.audit.record(r, ev)
	w.WriteHeader(http.StatusNoContent)
}

func (h environmentHandlers) extend(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	e, err := h.envs.Extend(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, envEvent(domain.AuditEnvironmentExtended, e))
	writeJSON(w, http.StatusOK, map[string]environmentView{"environment": toEnvView(e)})
}
