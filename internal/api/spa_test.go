package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newUIRouter(ui fstest.MapFS) http.Handler {
	return NewRouter(Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		UI:     ui,
	})
}

var builtUI = fstest.MapFS{
	"index.html":       {Data: []byte("<html>shell</html>")},
	"favicon.svg":      {Data: []byte("<svg/>")},
	"assets/app-1.js":  {Data: []byte("console.log(1)")},
	"assets/nested/.x": {Data: []byte("")},
}

func serve(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSPAServesAssets(t *testing.T) {
	rec := serve(newUIRouter(builtUI), "/assets/app-1.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Fatalf("got %d %q", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("assets should be cached immutably, got %q", rec.Header().Get("Cache-Control"))
	}
}

func TestSPAFallsBackToIndex(t *testing.T) {
	for _, p := range []string{"/", "/apps/shop", "/index.html", "/assets/nested", "/missing.js"} {
		rec := serve(newUIRouter(builtUI), p)
		if rec.Code != http.StatusOK || rec.Body.String() != "<html>shell</html>" {
			t.Errorf("%s: got %d %q", p, rec.Code, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: index must not be cached", p)
		}
	}
}

func TestSPADoesNotSwallowAPI(t *testing.T) {
	for _, p := range []string{"/api/v1/nope", "/api/other"} {
		rec := serve(newUIRouter(builtUI), p)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("%s: got %d %q", p, rec.Code, rec.Body)
		}
	}
}

func TestSPAWithoutBuild(t *testing.T) {
	rec := serve(newUIRouter(fstest.MapFS{".gitkeep": {}}), "/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d", rec.Code)
	}
}
