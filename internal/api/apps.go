package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/domain"
)

// AppService manages apps.
type AppService interface {
	List(ctx context.Context) ([]domain.App, error)
	Get(ctx context.Context, id int64) (domain.App, error)
	Create(ctx context.Context, in apps.CreateInput) (domain.App, error)
	Update(ctx context.Context, id int64, in apps.UpdateInput) (domain.App, error)
	Delete(ctx context.Context, id int64) (domain.App, error)
	RegenerateDeployKey(ctx context.Context, id int64) (domain.App, error)
}

// ConnectionTester checks repo access with an app's deploy key.
type ConnectionTester interface {
	Test(ctx context.Context, appID int64) (apps.ConnectionResult, error)
}

type appHandlers struct {
	logger *slog.Logger
	apps   AppService
	tester ConnectionTester
	audit  auditor
}

func (h appHandlers) register(rt *routes) {
	rt.handle("GET /apps", h.list)
	rt.handle("POST /apps", h.create)
	rt.handle("GET /apps/{id}", h.get)
	rt.handle("PATCH /apps/{id}", h.update)
	rt.handle("DELETE /apps/{id}", h.delete)
	rt.handle("POST /apps/{id}/deploy-key/regenerate", h.regenerateKey)
	if h.tester != nil {
		rt.handle("POST /apps/{id}/test-connection", h.testConnection)
	}
}

type appView struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	RepoURL         string    `json:"repoUrl"`
	DefaultBranch   string    `json:"defaultBranch"`
	DeployPublicKey string    `json:"deployPublicKey"`
	MaxEphemeral    int       `json:"maxEphemeral"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func toAppView(a domain.App) appView {
	return appView{
		ID: a.ID, Name: a.Name, RepoURL: a.RepoURL, DefaultBranch: a.DefaultBranch,
		DeployPublicKey: a.DeployPublicKey, MaxEphemeral: a.MaxEphemeral, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func appEvent(action string, a domain.App) domain.AuditEvent {
	return domain.AuditEvent{Action: action, TargetType: "app", TargetID: a.Name}
}

func (h appHandlers) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.apps.List(r.Context())
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]appView, 0, len(list))
	for _, a := range list {
		views = append(views, toAppView(a))
	}
	writeJSON(w, http.StatusOK, map[string][]appView{"apps": views})
}

type createAppRequest struct {
	Name          string `json:"name"`
	RepoURL       string `json:"repoUrl"`
	DefaultBranch string `json:"defaultBranch"`
}

func (h appHandlers) create(w http.ResponseWriter, r *http.Request) {
	var req createAppRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	a, err := h.apps.Create(r.Context(), apps.CreateInput(req))
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, appEvent(domain.AuditAppCreated, a))
	writeJSON(w, http.StatusCreated, map[string]appView{"app": toAppView(a)})
}

func (h appHandlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, err := h.apps.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]appView{"app": toAppView(a)})
}

type updateAppRequest struct {
	RepoURL       *string `json:"repoUrl"`
	DefaultBranch *string `json:"defaultBranch"`
	MaxEphemeral  *int    `json:"maxEphemeral"`
}

func (h appHandlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req updateAppRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	a, err := h.apps.Update(r.Context(), id, apps.UpdateInput(req))
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, appEvent(domain.AuditAppUpdated, a))
	writeJSON(w, http.StatusOK, map[string]appView{"app": toAppView(a)})
}

func (h appHandlers) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, err := h.apps.Delete(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, appEvent(domain.AuditAppDeleted, a))
	w.WriteHeader(http.StatusNoContent)
}

func (h appHandlers) regenerateKey(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, err := h.apps.RegenerateDeployKey(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, appEvent(domain.AuditAppDeployKeyRegenerated, a))
	writeJSON(w, http.StatusOK, map[string]appView{"app": toAppView(a)})
}

type connectionView struct {
	OK                  bool   `json:"ok"`
	DefaultBranchExists bool   `json:"defaultBranchExists"`
	BranchCount         int    `json:"branchCount"`
	Message             string `json:"message"`
}

func (h appHandlers) testConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	res, err := h.tester.Test(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]connectionView{"result": connectionView(res)})
}
