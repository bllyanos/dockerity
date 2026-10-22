// Command dockerity runs the Dockerity server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bllyanos/dockerity/internal/api"
	"github.com/bllyanos/dockerity/internal/config"
	"github.com/bllyanos/dockerity/internal/httpserver"
	"github.com/bllyanos/dockerity/internal/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "dockerity:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := logging.New(os.Stdout, cfg.LogLevel)

	router := api.NewRouter(api.Deps{Logger: logger})
	return httpserver.Run(ctx, logger, cfg.ListenAddr, router)
}
