package api

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestUserManagement(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	admin, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "admin")

	rec := bearer(h, http.MethodPost, APIPrefix+"/users", admin, `{"username":"bob","password":"long enough pw","isAdmin":false}`)
	if rec.Code != http.StatusCreated || strings.Contains(rec.Body.String(), "secret-hash") {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/users", admin, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"bob"`) {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/users/2/disable", admin, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"disabled":true`) {
		t.Fatalf("disable: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/users/2/enable", admin, ""); rec.Code != http.StatusOK {
		t.Fatalf("enable: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/users/2/password", admin, `{"password":"another long pw"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("reset: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/users/1/disable", admin, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("self disable: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/users/9/enable", admin, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown user: %d", rec.Code)
	}

	want := []string{domain.AuditTokenUsed, domain.AuditUserCreated, domain.AuditUserDisabled, domain.AuditUserEnabled, domain.AuditUserPasswordReset}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("audit %v, want %v", got, want)
	}
}

func TestUserManagementAdminOnly(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), domain.User{ID: 2, Username: "bob"}, "bob")
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/users", ""},
		{http.MethodPost, "/users", `{"username":"x","password":"long enough pw"}`},
		{http.MethodPost, "/users/1/disable", ""},
		{http.MethodPost, "/users/1/enable", ""},
		{http.MethodPost, "/users/1/password", `{"password":"long enough pw"}`},
	} {
		if rec := bearer(h, c.method, APIPrefix+c.path, raw, c.body); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s as non-admin: got %d", c.method, c.path, rec.Code)
		}
	}
}
