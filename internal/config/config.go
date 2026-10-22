// Package config loads Dockerity's runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
)

// Environment variable names.
const (
	EnvListenAddr    = "DOCKERITY_LISTEN_ADDR"
	EnvDataDir       = "DOCKERITY_DATA_DIR"
	EnvMasterKeyFile = "DOCKERITY_MASTER_KEY_FILE"
	EnvLogLevel      = "DOCKERITY_LOG_LEVEL"
)

// Defaults match the bootstrap compose file (docs/design.md section 13).
const (
	DefaultListenAddr    = ":8080"
	DefaultDataDir       = "/data"
	DefaultMasterKeyFile = "/run/secrets/master_key"
	DefaultLogLevel      = "info"
)

// Config is the validated runtime configuration.
type Config struct {
	ListenAddr    string
	DataDir       string
	MasterKeyFile string
	LogLevel      slog.Level
}

// LookupFunc has the signature of os.LookupEnv.
type LookupFunc func(key string) (string, bool)

// Load reads the configuration through lookup, applies defaults and validates it.
func Load(lookup LookupFunc) (Config, error) {
	get := func(key, def string) string {
		if v, ok := lookup(key); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return def
	}

	level, levelErr := parseLevel(get(EnvLogLevel, DefaultLogLevel))
	cfg := Config{
		ListenAddr:    get(EnvListenAddr, DefaultListenAddr),
		DataDir:       get(EnvDataDir, DefaultDataDir),
		MasterKeyFile: get(EnvMasterKeyFile, DefaultMasterKeyFile),
		LogLevel:      level,
	}

	var errs []error
	if levelErr != nil {
		errs = append(errs, levelErr)
	}
	if _, _, err := net.SplitHostPort(cfg.ListenAddr); err != nil {
		errs = append(errs, fmt.Errorf("%s: %w", EnvListenAddr, err))
	}
	return cfg, errors.Join(errs...)
}

func parseLevel(s string) (slog.Level, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo, fmt.Errorf("%s: unknown level %q", EnvLogLevel, s)
	}
	return l, nil
}
