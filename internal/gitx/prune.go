package gitx

import (
	"os"
	"path/filepath"
	"time"
)

// Prune removes the clones of apps not in keep that were not touched since
// before idle, so a clone made for an app created a moment ago survives.
// It returns the removed app names.
func (w *Workspace) Prune(keep map[string]bool, idle time.Time) ([]string, error) {
	entries, err := os.ReadDir(w.Root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, e := range entries {
		if !e.IsDir() || keep[e.Name()] {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.ModTime().Before(idle) {
			continue
		}
		dir := filepath.Join(w.Root, e.Name())
		unlock := w.lock(dir)
		err = os.RemoveAll(dir)
		unlock()
		if err != nil {
			return removed, err
		}
		removed = append(removed, e.Name())
	}
	return removed, nil
}
