package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type fakeSetup struct {
	done bool
}

func (f *fakeSetup) Needed(context.Context) (bool, error) { return !f.done, nil }

func (f *fakeSetup) Run(_ context.Context, username, pw string) (domain.User, error) {
	if f.done {
		return domain.User{}, domain.ErrConflict
	}
	if username == "" {
		return domain.User{}, domain.Invalid("username", "required")
	}
	f.done = true
	return domain.User{ID: 1, Username: username, PasswordHash: "secret-hash", IsAdmin: true, CreatedAt: time.Unix(0, 0).UTC()}, nil
}

func setupRouter(f *fakeSetup) http.Handler {
	return NewRouter(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Setup: f})
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestSetupFlow(t *testing.T) {
	h := setupRouter(&fakeSetup{})

	rec := do(h, http.MethodGet, "/api/v1/setup", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"needed":true}` {
		t.Fatalf("status: %d %s", rec.Code, rec.Body)
	}

	rec = do(h, http.MethodPost, "/api/v1/setup", `{"username":"admin","password":"long enough pw"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "secret-hash") {
		t.Fatal("password hash leaked in response")
	}
	var body struct{ User userView }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.User.Username != "admin" || !body.User.IsAdmin {
		t.Fatalf("unexpected body %s (%v)", rec.Body, err)
	}

	rec = do(h, http.MethodPost, "/api/v1/setup", `{"username":"again","password":"long enough pw"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second run: %d %s", rec.Code, rec.Body)
	}
}

func TestSetupBadInput(t *testing.T) {
	h := setupRouter(&fakeSetup{})
	cases := map[string]string{
		"validation":    `{"username":"","password":"x"}`,
		"unknown field": `{"username":"a","password":"b","admin":true}`,
		"not json":      `nope`,
		"trailing":      `{"username":"a","password":"b"} {}`,
	}
	for name, body := range cases {
		if rec := do(h, http.MethodPost, "/api/v1/setup", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d %s", name, rec.Code, rec.Body)
		}
	}
}
