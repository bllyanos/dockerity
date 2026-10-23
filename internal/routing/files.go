package routing

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Files writes one dynamic configuration file per environment into Dir, the
// directory Traefik's file provider watches.
type Files struct {
	Dir string
}

// path returns the file of app/env. Names are DNS labels, so they are safe in paths.
func (f Files) path(app, env string) string {
	return filepath.Join(f.Dir, fmt.Sprintf("dkt-%s-%s.yml", app, env))
}

// Write replaces the environment's file atomically, so Traefik never reads a
// half-written configuration.
func (f Files) Write(app, env string, data []byte) error {
	if err := os.MkdirAll(f.Dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(f.Dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), f.path(app, env))
}

// Remove deletes the environment's file; a missing file is not an error.
func (f Files) Remove(app, env string) error {
	err := os.Remove(f.path(app, env))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// Routed returns the container names listed as servers in the environment's
// file. A missing file means nothing is routed.
func (f Files) Routed(app, env string) (map[string]bool, error) {
	data, err := os.ReadFile(f.path(app, env))
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	return RoutedContainers(data)
}
