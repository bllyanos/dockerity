package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/deploy/drift"
)

// DriftService reports how an environment differs from its current deployment.
type DriftService interface {
	Check(ctx context.Context, envID int64) (drift.Report, error)
}

type driftHandlers struct {
	logger *slog.Logger
	drift  DriftService
}

func (h driftHandlers) register(rt *routes) {
	rt.handle("GET /environments/{id}/drift", h.get)
}

type driftItemView struct {
	Kind      string `json:"kind"`
	Service   string `json:"service"`
	Container string `json:"container"`
	Detail    string `json:"detail"`
}

type driftView struct {
	Items []driftItemView `json:"items"`
	// Leftovers counts stopped containers the reconciler will remove.
	Leftovers int `json:"leftovers"`
}

func (h driftHandlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	rep, err := h.drift.Check(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	items := make([]driftItemView, 0, len(rep.Items))
	for _, it := range rep.Items {
		items = append(items, driftItemView{Kind: string(it.Kind), Service: it.Service, Container: it.Container, Detail: it.Detail})
	}
	writeJSON(w, http.StatusOK, driftView{Items: items, Leftovers: len(rep.Remove)})
}
