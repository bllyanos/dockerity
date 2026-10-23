package main

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/api"
	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
	"github.com/bllyanos/dockerity/web"
)

// buildDeps assembles the services behind the HTTP API.
func buildDeps(logger *slog.Logger, db *sql.DB, docker *dockerx.Client) api.Deps {
	users := sqlite.Users{DB: db}
	return api.Deps{
		Logger: logger,
		UI:     web.Dist(),
		HealthChecks: []api.HealthCheck{
			sqlite.HealthCheck{DB: db},
			dockerx.HealthCheck{Docker: docker},
		},
		Setup: &auth.Setup{
			Users:          users,
			PasswordParams: password.DefaultParams,
			Now:            time.Now,
		},
		Audit: sqlite.AuditLog{DB: db},
		Tokens: &auth.Tokens{
			Users: users,
			Store: sqlite.APITokens{DB: db},
			Now:   time.Now,
		},
		Sessions: &auth.Sessions{
			Users:          users,
			Store:          sqlite.Sessions{DB: db},
			PasswordParams: password.DefaultParams,
			TTL:            auth.DefaultSessionTTL,
			Now:            time.Now,
		},
	}
}
