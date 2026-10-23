package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/deploy/drift"
	"github.com/bllyanos/dockerity/internal/domain"
)

type fakeDrift struct{}

func (fakeDrift) Check(_ context.Context, envID int64) (drift.Report, error) {
	if envID != 1 {
		return drift.Report{}, domain.ErrNotFound
	}
	return drift.Report{Items: []drift.Item{{Kind: drift.KindMissing, Service: "web", Detail: "no container"}}, Remove: []string{"x"}}, nil
}

func TestDriftAPI(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")
	rec := bearer(h, http.MethodGet, APIPrefix+"/environments/1/drift", raw, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `{"items":[{"kind":"missing","service":"web","container":"","detail":"no container"}],"leftovers":1}`) {
		t.Fatalf("drift: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/environments/9/drift", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
}
