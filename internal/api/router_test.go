package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeCheck struct {
	name string
	err  error
}

func (f fakeCheck) Name() string                    { return f.name }
func (f fakeCheck) Check(ctx context.Context) error { return f.err }

func newTestRouter(checks ...HealthCheck) http.Handler {
	return NewRouter(Deps{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		HealthChecks: checks,
	})
}

func get(t *testing.T, h http.Handler, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return rec, body
}

func TestHealthOK(t *testing.T) {
	rec, body := get(t, newTestRouter(fakeCheck{name: "db"}), "/healthz")
	if rec.Code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if body["checks"].(map[string]any)["db"] != "ok" {
		t.Fatalf("db check missing: %v", body)
	}
}

func TestHealthFailingCheck(t *testing.T) {
	rec, body := get(t, newTestRouter(fakeCheck{name: "db"}, fakeCheck{name: "docker", err: errors.New("down")}), "/healthz")
	if rec.Code != http.StatusServiceUnavailable || body["status"] != "unavailable" {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if body["checks"].(map[string]any)["docker"] != "down" {
		t.Fatalf("docker error missing: %v", body)
	}
}

func TestUnknownAPIRouteRequiresAuth(t *testing.T) {
	rec, body := get(t, newTestRouter(), APIPrefix+"/nope")
	if rec.Code != http.StatusUnauthorized || body["error"] != "unauthorized" {
		t.Fatalf("got %d %v", rec.Code, body)
	}
}
