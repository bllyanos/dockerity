package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestEnvContainersAPI(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")
	rec := bearer(h, http.MethodGet, APIPrefix+"/environments/1/containers", raw, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"dkt-shop-prod-web-d2","service":"web","deployment":"2","image":"","running":true,"routed":true`) {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/environments/9/containers", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
}
