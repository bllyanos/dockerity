package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/settings"
)

// SettingsService reads and updates instance settings.
type SettingsService interface {
	Get(ctx context.Context) (settings.Settings, error)
	Update(ctx context.Context, s settings.Settings) (settings.Settings, error)
}

type settingsHandlers struct {
	logger   *slog.Logger
	settings SettingsService
	audit    auditor
}

func (h settingsHandlers) register(rt *routes) {
	rt.handle("GET /settings", h.get)
	rt.handle("PUT /settings", adminOnly(h.update))
}

type settingsView struct {
	BaseDomain   string `json:"baseDomain"`
	TLSMode      string `json:"tlsMode"`
	CertFile     string `json:"certFile"`
	KeyFile      string `json:"keyFile"`
	CertResolver string `json:"certResolver"`
}

func toSettingsView(s settings.Settings) settingsView {
	return settingsView{BaseDomain: s.BaseDomain, TLSMode: string(s.TLSMode), CertFile: s.CertFile, KeyFile: s.KeyFile, CertResolver: s.CertResolver}
}

func (h settingsHandlers) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Get(r.Context())
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]settingsView{"settings": toSettingsView(s)})
}

func (h settingsHandlers) update(w http.ResponseWriter, r *http.Request) {
	var req settingsView
	if !decodeJSON(w, r, &req) {
		return
	}
	s, err := h.settings.Update(r.Context(), settings.Settings{
		BaseDomain: req.BaseDomain, TLSMode: settings.TLSMode(req.TLSMode),
		CertFile: req.CertFile, KeyFile: req.KeyFile, CertResolver: req.CertResolver,
	})
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, domain.AuditEvent{Action: domain.AuditSettingsUpdated, TargetType: "settings",
		Metadata: map[string]string{"base_domain": s.BaseDomain, "tls_mode": string(s.TLSMode)}})
	writeJSON(w, http.StatusOK, map[string]settingsView{"settings": toSettingsView(s)})
}
