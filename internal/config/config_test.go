package config

import (
	"log/slog"
	"strings"
	"testing"
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
		ListenAddr:    DefaultListenAddr,
		DataDir:       DefaultDataDir,
		MasterKeyFile: DefaultMasterKeyFile,
		LogLevel:      slog.LevelInfo,
	}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(lookupFrom(map[string]string{
		EnvListenAddr:    "127.0.0.1:9000",
		EnvDataDir:       " /tmp/dkt ",
		EnvMasterKeyFile: "/tmp/key",
		EnvLogLevel:      "DEBUG",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:9000" || cfg.DataDir != "/tmp/dkt" ||
		cfg.MasterKeyFile != "/tmp/key" || cfg.LogLevel != slog.LevelDebug {
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
		EnvListenAddr: "no-port",
		EnvLogLevel:   "loud",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, key := range []string{EnvListenAddr, EnvLogLevel} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}
