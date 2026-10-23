package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spawn"
)

// SpawnService spawns ephemeral environments.
type SpawnService interface {
	Spawn(ctx context.Context, appID int64, in spawn.Input, actor *int64) (domain.Environment, domain.Deployment, error)
}

type spawnHandlers struct {
	logger  *slog.Logger
	spawner SpawnService
	audit   auditor
}

func (h spawnHandlers) register(rt *routes) {
	rt.handle("POST /apps/{id}/spawn", h.create)
}

type spawnRequest struct {
	Name          string `json:"name"`
	BuildID       int64  `json:"buildId"`
	TemplateEnvID *int64 `json:"templateEnvId"`
	TTLHours      int    `json:"ttlHours"`
}

func (h spawnHandlers) create(w http.ResponseWriter, r *http.Request) {
	appID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req spawnRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	actor := p.User.ID
	env, d, err := h.spawner.Spawn(r.Context(), appID, spawn.Input{
		Name: req.Name, BuildID: req.BuildID, TemplateEnvID: req.TemplateEnvID, TTL: time.Duration(req.TTLHours) * time.Hour,
	}, &actor)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	ev := envEvent(domain.AuditEnvironmentSpawned, env)
	ev.Metadata["build_id"] = strconv.FormatInt(req.BuildID, 10)
	h.audit.record(r, ev)
	writeJSON(w, http.StatusAccepted, map[string]any{"environment": toEnvView(env), "deployment": toDeploymentView(d)})
}
