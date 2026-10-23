package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/deploy/warning"
	"github.com/bllyanos/dockerity/internal/domain"
)

// DeploymentService starts and reads deployments.
type DeploymentService interface {
	Start(ctx context.Context, envID, buildID int64, actor *int64) (domain.Deployment, error)
	List(ctx context.Context, envID int64) ([]domain.Deployment, error)
	Get(ctx context.Context, id int64) (domain.Deployment, []domain.DeploymentStep, error)
	Preview(ctx context.Context, envID, buildID int64) (warning.Warning, error)
}

type deploymentHandlers struct {
	logger      *slog.Logger
	deployments DeploymentService
	logs        LogSource
	audit       auditor
}

func (h deploymentHandlers) register(rt *routes) {
	rt.handle("POST /environments/{id}/deployments", h.start)
	rt.handle("POST /environments/{id}/deployments/preview", h.preview)
	rt.handle("GET /environments/{id}/deployments", h.list)
	rt.handle("GET /deployments/{id}", h.get)
	rt.handle("GET /deployments/{id}/logs", h.streamLogs)
}

type deploymentView struct {
	ID            int64                   `json:"id"`
	EnvID         int64                   `json:"envId"`
	BuildID       int64                   `json:"buildId"`
	SpecVersion   int                     `json:"specVersion"`
	SecretVersion int                     `json:"secretVersion"`
	Status        domain.DeploymentStatus `json:"status"`
	Current       bool                    `json:"current"`
	TriggeredBy   *int64                  `json:"triggeredBy"`
	Error         string                  `json:"error"`
	CreatedAt     time.Time               `json:"createdAt"`
	StartedAt     *time.Time              `json:"startedAt"`
	FinishedAt    *time.Time              `json:"finishedAt"`
}

type stepView struct {
	Position   int               `json:"position"`
	Kind       string            `json:"kind"`
	Service    string            `json:"service"`
	Status     domain.StepStatus `json:"status"`
	ExitCode   *int              `json:"exitCode"`
	Message    string            `json:"message"`
	StartedAt  *time.Time        `json:"startedAt"`
	FinishedAt *time.Time        `json:"finishedAt"`
}

func toDeploymentView(d domain.Deployment) deploymentView {
	return deploymentView{
		ID: d.ID, EnvID: d.EnvID, BuildID: d.BuildID, SpecVersion: d.SpecVersion, SecretVersion: d.SecretVersion,
		Status: d.Status, Current: d.Current, TriggeredBy: d.TriggeredBy, Error: d.Error,
		CreatedAt: d.CreatedAt, StartedAt: d.StartedAt, FinishedAt: d.FinishedAt,
	}
}

type startDeploymentRequest struct {
	BuildID int64 `json:"buildId"`
}

func (h deploymentHandlers) start(w http.ResponseWriter, r *http.Request) {
	envID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req startDeploymentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	actor := p.User.ID
	d, err := h.deployments.Start(r.Context(), envID, req.BuildID, &actor)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, domain.AuditEvent{
		Action: domain.AuditDeploymentStarted, TargetType: "deployment", TargetID: strconv.FormatInt(d.ID, 10),
		Metadata: map[string]string{
			"env_id": strconv.FormatInt(envID, 10), "build_id": strconv.FormatInt(d.BuildID, 10),
			"spec_version": strconv.Itoa(d.SpecVersion), "secret_version": strconv.Itoa(d.SecretVersion),
		},
	})
	writeJSON(w, http.StatusAccepted, map[string]deploymentView{"deployment": toDeploymentView(d)})
}

func (h deploymentHandlers) list(w http.ResponseWriter, r *http.Request) {
	envID, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := h.deployments.List(r.Context(), envID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]deploymentView, 0, len(list))
	for _, d := range list {
		views = append(views, toDeploymentView(d))
	}
	writeJSON(w, http.StatusOK, map[string][]deploymentView{"deployments": views})
}

func (h deploymentHandlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, steps, err := h.deployments.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]stepView, 0, len(steps))
	for _, s := range steps {
		views = append(views, stepView{Position: s.Position, Kind: s.Kind, Service: s.Service, Status: s.Status,
			ExitCode: s.ExitCode, Message: s.Message, StartedAt: s.StartedAt, FinishedAt: s.FinishedAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{"deployment": toDeploymentView(d), "steps": views})
}

func (h deploymentHandlers) streamLogs(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, _, err := h.deployments.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	serveLog(w, r, h.logger, h.logs, id, d.Status.Terminal())
}

type warningServiceView struct {
	Name         string                `json:"name"`
	Strategy     string                `json:"strategy"`
	Downtime     bool                  `json:"downtime"`
	OldImage     string                `json:"oldImage"`
	NewImage     string                `json:"newImage"`
	ImageChanged bool                  `json:"imageChanged"`
	MajorChange  bool                  `json:"majorChange"`
	Backup       bool                  `json:"backup"`
	LastBackup   map[string]*time.Time `json:"lastBackup"`
	Dependents   []string              `json:"dependents"`
}

type warningView struct {
	Downtime bool                 `json:"downtime"`
	Services []warningServiceView `json:"services"`
}

func (h deploymentHandlers) preview(w http.ResponseWriter, r *http.Request) {
	envID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req startDeploymentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	wn, err := h.deployments.Preview(r.Context(), envID, req.BuildID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	view := warningView{Downtime: wn.Downtime, Services: make([]warningServiceView, 0, len(wn.Services))}
	for _, s := range wn.Services {
		deps := s.Dependents
		if deps == nil {
			deps = []string{}
		}
		view.Services = append(view.Services, warningServiceView{
			Name: s.Name, Strategy: string(s.Strategy), Downtime: s.Downtime, OldImage: s.OldImage, NewImage: s.NewImage,
			ImageChanged: s.ImageChanged, MajorChange: s.MajorChange, Backup: s.Backup, LastBackup: s.LastBackup, Dependents: deps,
		})
	}
	writeJSON(w, http.StatusOK, map[string]warningView{"warning": view})
}
