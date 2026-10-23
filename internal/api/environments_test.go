package api

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestEnvironmentsAPI(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	rec := bearer(h, http.MethodPost, APIPrefix+"/apps/1/environments", raw, `{"name":"staging","domain":"staging.shop.test"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"type":"permanent"`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps/1/environments", raw, ""); !strings.Contains(rec.Body.String(), `"staging"`) {
		t.Fatalf("list: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodPatch, APIPrefix+"/environments/1", raw, `{"domain":"stg.shop.test"}`); !strings.Contains(rec.Body.String(), "stg.shop.test") {
		t.Fatalf("update: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/environments/1/extend", raw, ""); !strings.Contains(rec.Body.String(), `"expiresAt":"2030-01-01T00:00:00Z"`) {
		t.Fatalf("extend: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodPut, APIPrefix+"/environments/1/resources", raw, `{"resources":{"*":{"memory":"512m"}}}`); !strings.Contains(rec.Body.String(), `"resources":{"*":{"memory":"512m"}}`) {
		t.Fatalf("resources: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/environments/9", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodDelete, APIPrefix+"/environments/1", raw, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	want := []string{domain.AuditTokenUsed, domain.AuditEnvironmentCreated, domain.AuditEnvironmentUpdated, domain.AuditEnvironmentExtended, domain.AuditEnvironmentUpdated, domain.AuditEnvironmentDeleted}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("audit %v", got)
	}
}
