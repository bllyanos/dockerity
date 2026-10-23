package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func lookupFrom(m map[string]string) LookupFunc {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(lookupFrom(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Config{
		ListenAddr:      DefaultListenAddr,
		DataDir:         DefaultDataDir,
		MasterKeyFile:   DefaultMasterKeyFile,
		LogLevel:        slog.LevelInfo,
		BuildWorkers:    DefaultBuildWorkers,
		TraefikDir:      "/data/traefik",
		BackupsVolume:   DefaultBackupsVolume,
		LogRetention:    30 * 24 * time.Hour,
		BuildCacheBytes: 20 << 30,
	}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(lookupFrom(map[string]string{
		EnvListenAddr:       "127.0.0.1:9000",
		EnvDataDir:          " /tmp/dkt ",
		EnvMasterKeyFile:    "/tmp/key",
		EnvLogLevel:         "DEBUG",
		EnvBuildWorkers:     "3",
		EnvTraefikDir:       "/traefik",
		EnvLogRetentionDays: "0",
		EnvBuildCacheGB:     "5",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:9000" || cfg.DataDir != "/tmp/dkt" ||
		cfg.MasterKeyFile != "/tmp/key" || cfg.LogLevel != slog.LevelDebug || cfg.BuildWorkers != 3 || cfg.TraefikDir != "/traefik" ||
		cfg.LogRetention != 0 || cfg.BuildCacheBytes != 5<<30 {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
}

func TestLoadBlankFallsBackToDefault(t *testing.T) {
	cfg, err := Load(lookupFrom(map[string]string{EnvDataDir: "  "}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DataDir != DefaultDataDir {
		t.Fatalf("got %q, want default", cfg.DataDir)
	}
}

func TestLoadInvalid(t *testing.T) {
	_, err := Load(lookupFrom(map[string]string{
		EnvListenAddr:       "no-port",
		EnvLogLevel:         "loud",
		EnvBuildWorkers:     "0",
		EnvLogRetentionDays: "-1",
		EnvBuildCacheGB:     "lots",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, key := range []string{EnvListenAddr, EnvLogLevel, EnvBuildWorkers, EnvLogRetentionDays, EnvBuildCacheGB} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}
