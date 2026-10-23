// Command dockerity runs the Dockerity server.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bllyanos/dockerity/internal/api"
	"github.com/bllyanos/dockerity/internal/config"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/healthprobe"
	"github.com/bllyanos/dockerity/internal/httpserver"
	"github.com/bllyanos/dockerity/internal/logging"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
)

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "dockerity:", err)
		os.Exit(1)
	}
}

// dispatch runs the subcommand named in args, or the server by default.
func dispatch(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(args) == 0 {
		return run(ctx)
	}
	switch args[0] {
	case "healthcheck":
		return healthcheck(ctx)
	case "rotate-master-key":
		return rotateMasterKey(ctx, args[1:], os.Stdout)
	default:
		return fmt.Errorf("unknown command %q (commands: healthcheck, rotate-master-key)", args[0])
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := logging.New(os.Stdout, cfg.LogLevel)

	db, err := openDatabase(ctx, logger, cfg.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	keyring, err := openKeyring(ctx, logger, cfg.MasterKeyFile, db)
	if err != nil {
		return err
	}

	docker, err := connectDocker(ctx, logger)
	if err != nil {
		return err
	}
	defer docker.Close()

	svc := newServices(cfg, logger, db, docker, keyring)
	if err := svc.buildRunner.Start(ctx); err != nil {
		return fmt.Errorf("build runner: %w", err)
	}
	defer svc.buildRunner.Wait()

	router := api.NewRouter(svc.apiDeps())
	return httpserver.Run(ctx, logger, cfg.ListenAddr, router)
}

// healthcheck probes the local server; used by the container HEALTHCHECK.
func healthcheck(ctx context.Context) error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	url, err := healthprobe.URL(cfg.ListenAddr)
	if err != nil {
		return err
	}
	return healthprobe.Probe(ctx, url, 3*time.Second)
}

func openDatabase(ctx context.Context, logger *slog.Logger, dataDir string) (*sql.DB, error) {
	db, err := sqlite.Open(ctx, dataDir)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	migrations, err := sqlite.Migrations()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	applied, err := sqlite.Migrate(ctx, db, migrations)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	logger.Info("database ready", "path", filepath.Join(dataDir, sqlite.FileName), "migrations_applied", applied)
	return db, nil
}

// connectDocker fails fast when the Docker socket is not usable.
func connectDocker(ctx context.Context, logger *slog.Logger) (*dockerx.Client, error) {
	docker, err := dockerx.New()
	if err != nil {
		return nil, err
	}
	info, err := docker.ServerInfo(ctx)
	if err != nil {
		docker.Close()
		return nil, fmt.Errorf("docker engine unreachable (is /var/run/docker.sock mounted?): %w", err)
	}
	logger.Info("docker connected", "version", info.Version, "api_version", info.APIVersion, "os", info.OS, "arch", info.Arch)
	return docker, nil
}
