package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// BackupService takes, lists and restores backups.
type BackupService interface {
	List(ctx context.Context, envID int64) ([]domain.Backup, error)
	BackupService(ctx context.Context, envID int64, service string, reason domain.BackupReason) ([]domain.Backup, error)
	Restore(ctx context.Context, backupID int64) (domain.Backup, error)
}

type backupHandlers struct {
	logger  *slog.Logger
	backups BackupService
	audit   auditor
}

func (h backupHandlers) register(rt *routes) {
	rt.handle("GET /environments/{id}/backups", h.list)
	rt.handle("POST /environments/{id}/backups", h.create)
	rt.handle("POST /backups/{id}/restore", h.restore)
}

type backupView struct {
	ID        int64               `json:"id"`
	EnvID     *int64              `json:"envId"`
	AppName   string              `json:"appName"`
	EnvName   string              `json:"envName"`
	Service   string              `json:"service"`
	Volume    string              `json:"volume"`
	File      string              `json:"file"`
	SizeBytes int64               `json:"sizeBytes"`
	Reason    domain.BackupReason `json:"reason"`
	CreatedAt time.Time           `json:"createdAt"`
}

func toBackupView(b domain.Backup) backupView {
	return backupView{ID: b.ID, EnvID: b.EnvID, AppName: b.AppName, EnvName: b.EnvName, Service: b.Service,
		Volume: b.Volume, File: b.File, SizeBytes: b.SizeBytes, Reason: b.Reason, CreatedAt: b.CreatedAt}
}

func backupEvent(action string, b domain.Backup) domain.AuditEvent {
	return domain.AuditEvent{Action: action, TargetType: "backup", TargetID: strconv.FormatInt(b.ID, 10),
		Metadata: map[string]string{"volume": b.Volume, "file": b.File, "reason": string(b.Reason)}}
}

func (h backupHandlers) list(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := h.backups.List(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]backupView, 0, len(list))
	for _, b := range list {
		views = append(views, toBackupView(b))
	}
	writeJSON(w, http.StatusOK, map[string][]backupView{"backups": views})
}

type createBackupRequest struct {
	Service string `json:"service"`
}

func (h backupHandlers) create(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req createBackupRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	list, err := h.backups.BackupService(r.Context(), id, req.Service, domain.BackupManual)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]backupView, 0, len(list))
	for _, b := range list {
		h.audit.record(r, backupEvent(domain.AuditBackupCreated, b))
		views = append(views, toBackupView(b))
	}
	writeJSON(w, http.StatusCreated, map[string][]backupView{"backups": views})
}

func (h backupHandlers) restore(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	b, err := h.backups.Restore(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, backupEvent(domain.AuditBackupRestored, b))
	writeJSON(w, http.StatusOK, map[string]backupView{"backup": toBackupView(b)})
}
