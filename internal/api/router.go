// Package api holds the HTTP handlers and routing.
package api

import (
	"io/fs"
	"log/slog"
	"net/http"
)

// APIPrefix is the path prefix of the versioned JSON API.
const APIPrefix = "/api/v1"

// Deps are the collaborators the router needs.
type Deps struct {
	Logger       *slog.Logger
	HealthChecks []HealthCheck
	// UI is the built web app. When nil, no UI is served.
	UI fs.FS
	// Setup serves the first-run setup endpoints. When nil, they are not registered.
	Setup SetupService
}

// NewRouter builds the top-level HTTP handler.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler(d.HealthChecks))

	notFound := func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	}
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/", notFound)
	if d.Setup != nil {
		setupHandlers{logger: d.Logger, setup: d.Setup}.register(apiMux)
	}
	mux.Handle(APIPrefix+"/", http.StripPrefix(APIPrefix, apiMux))
	// Unknown API paths must not fall through to the UI.
	mux.HandleFunc("/api/", notFound)

	if d.UI != nil {
		mux.Handle("/", spaHandler(d.UI))
	}

	return logRequests(d.Logger, mux)
}
