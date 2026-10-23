package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestSpawnAPI(t *testing.T) {
	d := fullDeps()
	fs := d.Spawn.(*fakeSpawn)
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	rec := bearer(h, http.MethodPost, APIPrefix+"/apps/1/spawn", raw, `{"name":"","buildId":5,"templateEnvId":2,"ttlHours":24}`)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"type":"ephemeral"`) || !strings.Contains(rec.Body.String(), `"buildId":5`) {
		t.Fatalf("spawn: %d %s", rec.Code, rec.Body)
	}
	if fs.got.TTL != 24*time.Hour || fs.got.TemplateEnvID == nil || *fs.got.TemplateEnvID != 2 {
		t.Fatalf("input %+v", fs.got)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/apps/9/spawn", raw, `{"buildId":5}`); rec.Code != http.StatusNotFound {
		t.Fatalf("missing app: %d", rec.Code)
	}
	if got := log.actions(); got[len(got)-1] != domain.AuditEnvironmentSpawned {
		t.Fatalf("audit %v", got)
	}
}
