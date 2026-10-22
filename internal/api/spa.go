package api

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const indexFile = "index.html"

// spaHandler serves static files from ui and falls back to index.html for any
// other path, so client-side routes work on reload.
func spaHandler(ui fs.FS) http.Handler {
	files := http.FileServerFS(ui)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != indexFile && isFile(ui, name) {
			if strings.HasPrefix(name, "assets/") {
				// Vite fingerprints everything under assets/.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}
		serveIndex(w, r, ui)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, ui fs.FS) {
	body, err := fs.ReadFile(ui, indexFile)
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "UI not built: run `make web`", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		http.Error(w, "read index.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(body)
}

func isFile(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}
