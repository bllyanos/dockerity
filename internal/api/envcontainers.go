package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/environments"
)

// EnvContainerService lists an environment's containers.
type EnvContainerService interface {
	List(ctx context.Context, envID int64) ([]environments.Container, error)
}

type envContainerHandlers struct {
	logger     *slog.Logger
	containers EnvContainerService
}

func (h envContainerHandlers) register(rt *routes) {
	rt.handle("GET /environments/{id}/containers", h.list)
}

type containerView struct {
	Name       string `json:"name"`
	Service    string `json:"service"`
	Deployment string `json:"deployment"`
	Image      string `json:"image"`
	Running    bool   `json:"running"`
	Routed     bool   `json:"routed"`
}

func (h envContainerHandlers) list(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := h.containers.List(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]containerView, 0, len(list))
	for _, c := range list {
		views = append(views, containerView(c))
	}
	writeJSON(w, http.StatusOK, map[string][]containerView{"containers": views})
}
