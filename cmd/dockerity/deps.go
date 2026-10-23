package main

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/api"
	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/deploykey"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/gitx"
	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
	"github.com/bllyanos/dockerity/web"
)

// buildDeps assembles the services behind the HTTP API.
func buildDeps(logger *slog.Logger, db *sql.DB, docker *dockerx.Client, keyring *secrets.Keyring) api.Deps {
	users := sqlite.Users{DB: db}
	secretStore := &secrets.Builtin{Keys: keyring, Store: sqlite.SecretVersions{DB: db}, Now: time.Now}
	deployKeys := deploykey.Store{Secrets: secretStore}
	appService := &apps.Service{Store: sqlite.Apps{DB: db}, Keys: deployKeys, Now: time.Now}
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
		Apps:  appService,
		Specs: &apps.Specs{Apps: appService, Store: sqlite.SpecVersions{DB: db}, Now: time.Now},
		ConnectionTester: &apps.ConnectionTester{
			Apps: appService,
			Keys: deployKeys,
			Git:  &gitx.Client{HostKeys: sqlite.KnownHosts{DB: db}, Now: time.Now},
		},
		Users: &auth.UserAdmin{
			Users:          users,
			Credentials:    credentials{sqlite.Sessions{DB: db}, sqlite.APITokens{DB: db}},
			PasswordParams: password.DefaultParams,
			Now:            time.Now,
		},
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

// credentials combines the stores that end a user's sessions and API tokens.
type credentials struct {
	sqlite.Sessions
	sqlite.APITokens
}
