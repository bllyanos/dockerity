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
	// Tokens manages API tokens. When nil, bearer auth is rejected and token routes are not registered.
	Tokens TokenService
	// Users manages users (admin only). When nil, user routes are not registered.
	Users UserAdminService
	// Apps manages apps. When nil, app routes are not registered.
	Apps AppService
	// Specs saves and reads app spec versions. When nil, spec routes are not registered.
	Specs SpecService
	// Environments manages environments. When nil, environment routes are not registered.
	Environments EnvironmentService
	// EnvContainers lists environment containers. When nil, the route is not registered.
	EnvContainers EnvContainerService
	// EnvSecrets manages environment secrets. When nil, secret routes are not registered.
	EnvSecrets EnvSecretService
	// Deployments starts and reads deployments. When nil, deployment routes are not registered.
	Deployments DeploymentService
	// DeploymentLogs serves persisted and live deployment logs.
	DeploymentLogs LogSource
	// Builds triggers and reads builds. When nil, build routes are not registered.
	Builds BuildService
	// BuildLogs serves persisted and live build logs.
	BuildLogs LogSource
	// ConnectionTester tests repo access. When nil, the test-connection route is not registered.
	ConnectionTester ConnectionTester
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
	guard := authGuard{logger: d.Logger, routes: rt, sessions: d.Sessions, tokens: d.Tokens, audit: newAuditor(d)}
	mux.Handle(APIPrefix+"/", http.StripPrefix(APIPrefix, guard))
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
	aud := newAuditor(d)
	if d.Audit != nil {
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
	if d.Users != nil {
		userHandlers{logger: d.Logger, users: d.Users, audit: aud}.register(rt)
	}
	if d.Apps != nil {
		appHandlers{logger: d.Logger, apps: d.Apps, tester: d.ConnectionTester, audit: aud}.register(rt)
	}
	if d.Specs != nil {
		specHandlers{logger: d.Logger, specs: d.Specs, audit: aud}.register(rt)
	}
	if d.Environments != nil {
		environmentHandlers{logger: d.Logger, envs: d.Environments, audit: aud}.register(rt)
	}
	if d.EnvContainers != nil {
		envContainerHandlers{logger: d.Logger, containers: d.EnvContainers}.register(rt)
	}
	if d.EnvSecrets != nil {
		envSecretHandlers{logger: d.Logger, secrets: d.EnvSecrets, audit: aud}.register(rt)
	}
	if d.Deployments != nil {
		deploymentHandlers{logger: d.Logger, deployments: d.Deployments, logs: d.DeploymentLogs, audit: aud}.register(rt)
	}
	if d.Builds != nil {
		buildHandlers{logger: d.Logger, builds: d.Builds, logs: d.BuildLogs, audit: aud}.register(rt)
	}
	if d.Tokens != nil {
		tokenHandlers{logger: d.Logger, tokens: d.Tokens, audit: aud}.register(rt)
	}
	return rt
}

func newAuditor(d Deps) auditor {
	a := auditor{logger: d.Logger}
	if d.Audit != nil {
		a.rec = d.Audit
	}
	return a
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}
