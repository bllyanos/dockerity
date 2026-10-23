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

// SpecService saves and reads app spec versions.
type SpecService interface {
	Save(ctx context.Context, appID int64, yaml string, actor *int64) (domain.AppSpecVersion, bool, error)
	Latest(ctx context.Context, appID int64) (domain.AppSpecVersion, error)
	Get(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error)
	List(ctx context.Context, appID int64) ([]domain.AppSpecVersion, error)
	Diff(ctx context.Context, appID int64, from, to int) (string, error)
}

type specHandlers struct {
	logger *slog.Logger
	specs  SpecService
	audit  auditor
}

func (h specHandlers) register(rt *routes) {
	rt.handle("GET /apps/{id}/spec", h.latest)
	rt.handle("PUT /apps/{id}/spec", h.save)
	rt.handle("GET /apps/{id}/spec/versions", h.list)
	rt.handle("GET /apps/{id}/spec/versions/{version}", h.get)
	rt.handle("GET /apps/{id}/spec/diff", h.diff)
}

type specVersionView struct {
	Version   int       `json:"version"`
	YAML      string    `json:"yaml,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy *int64    `json:"createdBy"`
}

func toSpecView(v domain.AppSpecVersion) specVersionView {
	return specVersionView{Version: v.Version, YAML: v.YAML, CreatedAt: v.CreatedAt, CreatedBy: v.CreatedBy}
}

func (h specHandlers) latest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	v, err := h.specs.Latest(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]specVersionView{"spec": toSpecView(v)})
}

type saveSpecRequest struct {
	YAML string `json:"yaml"`
}

func (h specHandlers) save(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req saveSpecRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	actor := p.User.ID
	v, created, err := h.specs.Save(r.Context(), id, req.YAML, &actor)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusCreated
		h.audit.record(r, domain.AuditEvent{
			Action: domain.AuditAppSpecUpdated, TargetType: "app", TargetID: strconv.FormatInt(id, 10),
			Metadata: map[string]string{"version": strconv.Itoa(v.Version)},
		})
	}
	writeJSON(w, code, map[string]any{"spec": toSpecView(v), "created": created})
}

func (h specHandlers) list(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	versions, err := h.specs.List(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]specVersionView, 0, len(versions))
	for _, v := range versions {
		v.YAML = "" // the list is metadata only; fetch a version for its content
		views = append(views, toSpecView(v))
	}
	writeJSON(w, http.StatusOK, map[string][]specVersionView{"versions": views})
}

func (h specHandlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "version: must be an integer")
		return
	}
	v, err := h.specs.Get(r.Context(), id, version)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]specVersionView{"spec": toSpecView(v)})
}

func (h specHandlers) diff(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	from, err1 := queryInt(r, "from", 0)
	to, err2 := queryInt(r, "to", 0)
	if err1 != nil || err2 != nil || from < 1 || to < 1 {
		writeError(w, http.StatusBadRequest, "from and to must be version numbers")
		return
	}
	d, err := h.specs.Diff(r.Context(), id, int(from), int(to))
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"diff": d})
}
