package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/domain"
)

// BuildService triggers, lists and cancels builds.
type BuildService interface {
	Trigger(ctx context.Context, appID int64, ref string, actor *int64) (domain.Build, error)
	List(ctx context.Context, appID int64) ([]domain.Build, error)
	Get(ctx context.Context, id int64) (domain.Build, []domain.BuildImage, error)
	Cancel(ctx context.Context, id int64) (domain.Build, error)
}

// BuildLogs gives access to persisted and live build logs.
type BuildLogs struct {
	Store buildlog.Store
	Hub   *buildlog.Hub
}

type buildHandlers struct {
	logger *slog.Logger
	builds BuildService
	logs   BuildLogs
	audit  auditor
}

func (h buildHandlers) register(rt *routes) {
	rt.handle("POST /apps/{id}/builds", h.trigger)
	rt.handle("GET /apps/{id}/builds", h.list)
	rt.handle("GET /builds/{id}", h.get)
	rt.handle("POST /builds/{id}/cancel", h.cancel)
	rt.handle("GET /builds/{id}/logs", h.streamLogs)
}

type buildView struct {
	ID          int64              `json:"id"`
	AppID       int64              `json:"appId"`
	Ref         string             `json:"ref"`
	CommitSHA   string             `json:"commitSha"`
	Status      domain.BuildStatus `json:"status"`
	TriggeredBy *int64             `json:"triggeredBy"`
	Error       string             `json:"error"`
	CreatedAt   time.Time          `json:"createdAt"`
	StartedAt   *time.Time         `json:"startedAt"`
	FinishedAt  *time.Time         `json:"finishedAt"`
}

type buildImageView struct {
	Service  string `json:"service"`
	ImageTag string `json:"imageTag"`
	ImageID  string `json:"imageId"`
}

func toBuildView(b domain.Build) buildView {
	return buildView{
		ID: b.ID, AppID: b.AppID, Ref: b.Ref, CommitSHA: b.CommitSHA, Status: b.Status,
		TriggeredBy: b.TriggeredBy, Error: b.Error, CreatedAt: b.CreatedAt, StartedAt: b.StartedAt, FinishedAt: b.FinishedAt,
	}
}

func buildEvent(action string, b domain.Build) domain.AuditEvent {
	return domain.AuditEvent{
		Action: action, TargetType: "build", TargetID: strconv.FormatInt(b.ID, 10),
		Metadata: map[string]string{"app_id": strconv.FormatInt(b.AppID, 10), "ref": b.Ref},
	}
}

type triggerBuildRequest struct {
	Ref string `json:"ref"`
}

func (h buildHandlers) trigger(w http.ResponseWriter, r *http.Request) {
	appID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req triggerBuildRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	actor := p.User.ID
	b, err := h.builds.Trigger(r.Context(), appID, req.Ref, &actor)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, buildEvent(domain.AuditBuildStarted, b))
	writeJSON(w, http.StatusAccepted, map[string]buildView{"build": toBuildView(b)})
}

func (h buildHandlers) list(w http.ResponseWriter, r *http.Request) {
	appID, ok := pathID(w, r)
	if !ok {
		return
	}
	builds, err := h.builds.List(r.Context(), appID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]buildView, 0, len(builds))
	for _, b := range builds {
		views = append(views, toBuildView(b))
	}
	writeJSON(w, http.StatusOK, map[string][]buildView{"builds": views})
}

func (h buildHandlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	b, imgs, err := h.builds.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	images := make([]buildImageView, 0, len(imgs))
	for _, i := range imgs {
		images = append(images, buildImageView{Service: i.Service, ImageTag: i.ImageTag, ImageID: i.ImageID})
	}
	writeJSON(w, http.StatusOK, map[string]any{"build": toBuildView(b), "images": images})
}

func (h buildHandlers) cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	b, err := h.builds.Cancel(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, buildEvent(domain.AuditBuildCanceled, b))
	w.WriteHeader(http.StatusAccepted)
}
