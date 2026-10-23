// Package api holds the HTTP handlers and routing.
package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/bllyanos/dockerity/internal/auth/ratelimit"
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
	// Sessions authenticates requests. When nil, every protected route returns 401.
	Sessions SessionService
	// Audit stores the audit log. When nil, nothing is recorded and the audit API is not registered.
	Audit AuditLog
	// LoginLimiter throttles failed logins. Defaults to 10 failures per 15 minutes per user and per IP.
	LoginLimiter *ratelimit.Limiter
}

// NewRouter builds the top-level HTTP handler.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler(d.HealthChecks))

	rt := apiRoutes(d)
	mux.Handle(APIPrefix+"/", http.StripPrefix(APIPrefix, requireAuth(d.Logger, rt, d.Sessions)))
	// Unknown API paths must not fall through to the UI.
	mux.HandleFunc("/api/", notFound)

	if d.UI != nil {
		mux.Handle("/", spaHandler(d.UI))
	}

	csrf := http.NewCrossOriginProtection()
	return logRequests(d.Logger, csrf.Handler(mux))
}

// apiRoutes registers every /api/v1 route. Routes are protected unless
// registered with handlePublic.
func apiRoutes(d Deps) *routes {
	rt := newRoutes()
	rt.handle("/", notFound)
	aud := auditor{logger: d.Logger}
	if d.Audit != nil {
		aud.rec = d.Audit
		auditHandlers{logger: d.Logger, log: d.Audit}.register(rt)
	}

	if d.Setup != nil {
		setupHandlers{logger: d.Logger, setup: d.Setup, sessions: d.Sessions, audit: aud}.register(rt)
	}
	if d.Sessions != nil {
		limiter := d.LoginLimiter
		if limiter == nil {
			limiter = ratelimit.New(10, 15*time.Minute)
		}
		authHandlers{logger: d.Logger, sessions: d.Sessions, limiter: limiter, audit: aud}.register(rt)
	}
	return rt
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}
