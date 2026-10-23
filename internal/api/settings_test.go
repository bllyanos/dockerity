package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestSettingsAPI(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	admin, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "a")
	user, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), domain.User{ID: 2, Username: "bob"}, "b")

	if rec := bearer(h, http.MethodGet, APIPrefix+"/settings", user, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"tlsMode":"none"`) {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	body := `{"baseDomain":"apps.example.com","tlsMode":"resolver","certFile":"","keyFile":"","certResolver":"le"}`
	if rec := bearer(h, http.MethodPut, APIPrefix+"/settings", user, body); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin update: %d", rec.Code)
	}
	if rec := bearer(h, http.MethodPut, APIPrefix+"/settings", admin, body); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"baseDomain":"apps.example.com"`) {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if rec := bearer(h, http.MethodPut, APIPrefix+"/settings", admin, `{"baseDomain":"x","tlsMode":"none","certFile":"","keyFile":"","certResolver":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid: %d", rec.Code)
	}
}
