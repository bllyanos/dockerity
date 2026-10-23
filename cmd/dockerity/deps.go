package main

import (
	"database/sql"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/bllyanos/dockerity/internal/api"
	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/auth/password"
	"github.com/bllyanos/dockerity/internal/backups"
	"github.com/bllyanos/dockerity/internal/build"
	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/config"
	"github.com/bllyanos/dockerity/internal/deploy/executor"
	"github.com/bllyanos/dockerity/internal/deploykey"
	"github.com/bllyanos/dockerity/internal/deployments"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/environments"
	"github.com/bllyanos/dockerity/internal/envsecrets"
	"github.com/bllyanos/dockerity/internal/gitx"
	"github.com/bllyanos/dockerity/internal/routing"
	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/settings"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
	"github.com/bllyanos/dockerity/web"
)

// services holds Dockerity's use cases, built once and shared by the HTTP API
// and background workers.
type services struct {
	logger       *slog.Logger
	db           *sql.DB
	docker       *dockerx.Client
	users        sqlite.Users
	apps         *apps.Service
	specs        *apps.Specs
	secrets      secrets.SecretStore
	envs         *environments.Service
	deployKeys   deploykey.Store
	git          *gitx.Client
	buildLogs    buildlog.Store
	buildHub     *buildlog.Hub
	buildRunner  *build.Runner
	deployLogs   buildlog.Store
	deployHub    *buildlog.Hub
	deployRunner *executor.Runner
	routes       routing.Files
	backups      *backups.Service
	settings     *settings.Service
}

func newServices(cfg config.Config, logger *slog.Logger, db *sql.DB, docker *dockerx.Client, keyring *secrets.Keyring) *services {
	s := &services{logger: logger, db: db, docker: docker, users: sqlite.Users{DB: db}, routes: routing.Files{Dir: cfg.TraefikDir}}
	s.settings = &settings.Service{Store: sqlite.Settings{DB: db}, Now: time.Now}
	s.secrets = &secrets.Builtin{Keys: keyring, Store: sqlite.SecretVersions{DB: db}, Now: time.Now}
	s.deployKeys = deploykey.Store{Secrets: s.secrets}
	s.apps = &apps.Service{Store: sqlite.Apps{DB: db}, Keys: s.deployKeys, Now: time.Now}
	s.specs = &apps.Specs{Apps: s.apps, Store: sqlite.SpecVersions{DB: db}, Now: time.Now}
	s.envs = &environments.Service{
		Store:       sqlite.Environments{DB: db},
		Apps:        s.apps,
		Deployments: sqlite.Deployments{DB: db},
		Runtime:     executor.Teardown{Docker: docker, Routes: s.routes},
		Now:         time.Now,
	}
	s.git = &gitx.Client{HostKeys: sqlite.KnownHosts{DB: db}, Now: time.Now}
	s.buildLogs = buildlog.Store{Dir: filepath.Join(cfg.DataDir, "logs", "builds")}
	s.buildHub = buildlog.NewHub()
	s.buildRunner = &build.Runner{
		Builds:  sqlite.Builds{DB: db},
		Apps:    s.apps,
		Specs:   s.specs,
		Keys:    s.deployKeys,
		Git:     &gitx.Workspace{Root: filepath.Join(cfg.DataDir, "workspace"), Client: s.git},
		Docker:  docker,
		Logs:    s.buildLogs,
		Hub:     s.buildHub,
		Logger:  logger.With("component", "build"),
		Now:     time.Now,
		Workers: cfg.BuildWorkers,
	}
	s.deployLogs = buildlog.Store{Dir: filepath.Join(cfg.DataDir, "logs", "deployments")}
	s.deployHub = buildlog.NewHub()
	s.deployRunner = &executor.Runner{Exec: &executor.Executor{
		Store:   sqlite.Deployments{DB: db},
		Envs:    s.envs,
		Specs:   s.specs,
		Builds:  sqlite.Builds{DB: db},
		Secrets: s.secrets,
		Docker:  docker,
		Logs:    s.deployLogs,
		Hub:     s.deployHub,
		Routing: routing.DefaultOptions,
		Routes:  s.routes,
		AllEnvs: sqlite.Environments{DB: db},
		Logger:  logger.With("component", "deploy"),
		Now:     time.Now,
	}}
	s.envs.Routes = s.deployRunner.Exec
	s.backups = &backups.Service{
		Envs:          s.envs,
		Store:         sqlite.Backups{DB: db},
		Docker:        docker,
		Specs:         backups.CurrentSpecs{Deployments: sqlite.Deployments{DB: db}, Specs: s.specs},
		BackupsVolume: cfg.BackupsVolume,
		Now:           time.Now,
	}
	s.deployRunner.Exec.Backups = s.backups
	s.envs.Backups = s.backups
	return s
}

// apiDeps assembles the collaborators of the HTTP API.
func (s *services) apiDeps() api.Deps {
	db := s.db
	return api.Deps{
		Logger: s.logger,
		UI:     web.Dist(),
		HealthChecks: []api.HealthCheck{
			sqlite.HealthCheck{DB: db},
			dockerx.HealthCheck{Docker: s.docker},
		},
		Setup: &auth.Setup{
			Users:          s.users,
			PasswordParams: password.DefaultParams,
			Now:            time.Now,
		},
		Audit: sqlite.AuditLog{DB: db},
		Apps:  s.apps,
		Specs: s.specs,
		Builds: &build.Service{
			Builds: sqlite.Builds{DB: db},
			Apps:   s.apps,
			Queue:  s.buildRunner,
			Now:    time.Now,
		},
		BuildLogs: api.LogSource{Store: s.buildLogs, Hub: s.buildHub},
		Deployments: &deployments.Service{
			Store:     sqlite.Deployments{DB: db},
			Envs:      s.envs,
			Builds:    sqlite.Builds{DB: db},
			Specs:     s.specs,
			Secrets:   s.secrets,
			Runner:    s.deployRunner,
			Previewer: s.deployRunner.Exec,
			Backups:   sqlite.Backups{DB: db},
			Now:       time.Now,
		},
		DeploymentLogs: api.LogSource{Store: s.deployLogs, Hub: s.deployHub},
		Environments:   s.envs,
		Backups:        s.backups,
		Settings:       s.settings,
		EnvContainers:  &environments.Containers{Envs: s.envs, Docker: s.docker, Routes: s.routes},
		EnvSecrets:     &envsecrets.Service{Envs: s.envs, Secrets: s.secrets},
		ConnectionTester: &apps.ConnectionTester{
			Apps: s.apps,
			Keys: s.deployKeys,
			Git:  s.git,
		},
		Users: &auth.UserAdmin{
			Users:          s.users,
			Credentials:    credentials{sqlite.Sessions{DB: db}, sqlite.APITokens{DB: db}},
			PasswordParams: password.DefaultParams,
			Now:            time.Now,
		},
		Tokens: &auth.Tokens{
			Users: s.users,
			Store: sqlite.APITokens{DB: db},
			Now:   time.Now,
		},
		Sessions: &auth.Sessions{
			Users:          s.users,
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
