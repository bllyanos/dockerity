package api

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestAppsAPI(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	rec := bearer(h, http.MethodPost, APIPrefix+"/apps", raw, `{"name":"shop","repoUrl":"git@github.com:org/shop.git","defaultBranch":"main"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"deployPublicKey":"ssh-ed25519 AAAA"`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/apps", raw, `{"name":"shop","repoUrl":"x"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/apps", raw, `{"name":"","repoUrl":"x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps", raw, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"shop"`) {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPatch, APIPrefix+"/apps/1", raw, `{"defaultBranch":"develop"}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"develop"`) {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps/9", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodDelete, APIPrefix+"/apps/1", raw, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}

	want := []string{domain.AuditTokenUsed, domain.AuditAppCreated, domain.AuditAppUpdated, domain.AuditAppDeleted}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("audit %v, want %v", got, want)
	}
}
