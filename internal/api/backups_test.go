package api

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestBackupsAPI(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	rec := bearer(h, http.MethodPost, APIPrefix+"/environments/1/backups", raw, `{"service":"db"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"reason":"manual"`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/environments/1/backups", raw, `{"service":"web"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("non-stateful: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/environments/1/backups", raw, ""); !strings.Contains(rec.Body.String(), `"volume":"dkt-shop-prod-db-data"`) {
		t.Fatalf("list: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/backups/1/restore", raw, ""); rec.Code != http.StatusOK {
		t.Fatalf("restore: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/backups/9/restore", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	want := []string{domain.AuditTokenUsed, domain.AuditBackupCreated, domain.AuditBackupRestored}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("audit %v", got)
	}
}
