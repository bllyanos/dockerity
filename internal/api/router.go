// Package api holds the HTTP handlers and routing.
package api

import (
	"log/slog"
	"net/http"
)

// APIPrefix is the path prefix of the versioned JSON API.
const APIPrefix = "/api/v1"

// Deps are the collaborators the router needs.
type Deps struct {
	Logger       *slog.Logger
	HealthChecks []HealthCheck
}

// NewRouter builds the top-level HTTP handler.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler(d.HealthChecks))

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.Handle(APIPrefix+"/", http.StripPrefix(APIPrefix, apiMux))

	return logRequests(d.Logger, mux)
}
