package api

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestDeploymentsAPI(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	rec := bearer(h, http.MethodPost, APIPrefix+"/environments/1/deployments", raw, `{"buildId":5}`)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"buildId":5`) {
		t.Fatalf("start: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/environments/1/deployments", raw, `{"buildId":5}`); rec.Code != http.StatusConflict {
		t.Fatalf("second deploy: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/environments/1/deployments", raw, ""); !strings.Contains(rec.Body.String(), `"status":"pending"`) {
		t.Fatalf("list: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/deployments/1", raw, ""); !strings.Contains(rec.Body.String(), `"kind":"ensure_network"`) {
		t.Fatalf("get: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/deployments/9", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if got := log.actions(); !slices.Equal(got, []string{domain.AuditTokenUsed, domain.AuditDeploymentStarted}) {
		t.Fatalf("audit %v", got)
	}
}

func TestDeploymentPreviewAPI(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")
	rec := bearer(h, http.MethodPost, APIPrefix+"/environments/1/deployments/preview", raw, `{"buildId":5}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"majorChange":true`) || !strings.Contains(rec.Body.String(), `"dependents":["api"]`) {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/environments/1/deployments/preview", raw, `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing build: %d", rec.Code)
	}
}

func TestRollbackAPI(t *testing.T) {
	d := fullDeps()
	fd := d.Deployments.(*fakeDeployments)
	fd.deps = []domain.Deployment{{ID: 1, EnvID: 1, BuildID: 7, SpecVersion: 2, Status: domain.DeploymentSucceeded}}
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	if rec := bearer(h, http.MethodPost, APIPrefix+"/deployments/1/rollback/preview", raw, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"warning"`) {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body)
	}
	rec := bearer(h, http.MethodPost, APIPrefix+"/deployments/1/rollback", raw, "")
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"buildId":7`) {
		t.Fatalf("rollback: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/deployments/9/rollback", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if got := log.actions(); got[len(got)-1] != domain.AuditDeploymentRollback {
		t.Fatalf("audit %v", got)
	}
}
