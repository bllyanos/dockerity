package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// AuditLog stores and lists audit events.
type AuditLog interface {
	Record(ctx context.Context, e domain.AuditEvent) error
	ListAudit(ctx context.Context, before int64, limit int) ([]domain.AuditEvent, error)
}

const (
	defaultAuditPage = 50
	maxAuditPage     = 200
)

type auditHandlers struct {
	logger *slog.Logger
	log    AuditLog
}

func (h auditHandlers) register(rt *routes) {
	rt.handle("GET /audit", adminOnly(h.list))
}

type auditEventView struct {
	ID            int64             `json:"id"`
	At            time.Time         `json:"at"`
	ActorUserID   *int64            `json:"actorUserId"`
	ActorUsername string            `json:"actorUsername"`
	ActorTokenID  *int64            `json:"actorTokenId"`
	Action        string            `json:"action"`
	TargetType    string            `json:"targetType"`
	TargetID      string            `json:"targetId"`
	Metadata      map[string]string `json:"metadata"`
	RemoteAddr    string            `json:"remoteAddr"`
}

type auditPage struct {
	Events []auditEventView `json:"events"`
	// NextBefore is the cursor for the next (older) page, or null at the end.
	NextBefore *int64 `json:"nextBefore"`
}

func (h auditHandlers) list(w http.ResponseWriter, r *http.Request) {
	before, err := queryInt(r, "before", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := queryInt(r, "limit", defaultAuditPage)
	if err != nil || limit < 1 || limit > maxAuditPage {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and "+strconv.Itoa(maxAuditPage))
		return
	}

	events, err := h.log.ListAudit(r.Context(), before, int(limit))
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	page := auditPage{Events: make([]auditEventView, 0, len(events))}
	for _, e := range events {
		page.Events = append(page.Events, auditEventView{
			ID: e.ID, At: e.At, ActorUserID: e.ActorUserID, ActorUsername: e.ActorUsername,
			ActorTokenID: e.ActorTokenID, Action: e.Action, TargetType: e.TargetType,
			TargetID: e.TargetID, Metadata: e.Metadata, RemoteAddr: e.RemoteAddr,
		})
	}
	if len(events) == int(limit) {
		last := events[len(events)-1].ID
		page.NextBefore = &last
	}
	writeJSON(w, http.StatusOK, page)
}

// queryInt parses an optional integer query parameter.
func queryInt(r *http.Request, name string, def int64) (int64, error) {
	s := r.URL.Query().Get(name)
	if s == "" {
		return def, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, domain.Invalid(name, "must be an integer")
	}
	return v, nil
}
