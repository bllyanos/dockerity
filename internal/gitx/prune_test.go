package gitx

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestPrune(t *testing.T) {
	ws := &Workspace{Root: t.TempDir()}
	old := time.Now().Add(-2 * time.Hour)
	for _, app := range []string{"shop", "gone", "fresh"} {
		dir := filepath.Join(ws.Root, app)
		_ = os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
		if app != "fresh" {
			_ = os.Chtimes(dir, old, old)
		}
	}
	_ = os.WriteFile(filepath.Join(ws.Root, "stray.txt"), nil, 0o600)

	removed, err := ws.Prune(map[string]bool{"shop": true}, time.Now().Add(-time.Hour))
	if err != nil || !slices.Equal(removed, []string{"gone"}) {
		t.Fatalf("removed %v %v", removed, err)
	}
	for _, name := range []string{"shop", "fresh", "stray.txt"} {
		if _, err := os.Stat(filepath.Join(ws.Root, name)); err != nil {
			t.Fatalf("%s removed", name)
		}
	}
	if removed, err := (&Workspace{Root: filepath.Join(ws.Root, "none")}).Prune(nil, time.Now()); err != nil || len(removed) != 0 {
		t.Fatalf("missing root: %v %v", removed, err)
	}
}
