package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/domain"
)

type memSpecStore struct{ versions []domain.AppSpecVersion }

func (m *memSpecStore) InsertSpecVersion(_ context.Context, v domain.AppSpecVersion) error {
	m.versions = append(m.versions, v)
	return nil
}

func (m *memSpecStore) LatestSpecVersion(_ context.Context, appID int64) (domain.AppSpecVersion, error) {
	for i := len(m.versions) - 1; i >= 0; i-- {
		if m.versions[i].AppID == appID {
			return m.versions[i], nil
		}
	}
	return domain.AppSpecVersion{}, domain.ErrNotFound
}

func (m *memSpecStore) GetSpecVersion(_ context.Context, appID int64, v int) (domain.AppSpecVersion, error) {
	for _, x := range m.versions {
		if x.AppID == appID && x.Version == v {
			return x, nil
		}
	}
	return domain.AppSpecVersion{}, domain.ErrNotFound
}

func (m *memSpecStore) ListSpecVersions(context.Context, int64) ([]domain.AppSpecVersion, error) {
	return m.versions, nil
}

func specRouter(t *testing.T) (http.Handler, string, *fakeAudit) {
	t.Helper()
	d := fullDeps()
	fa := d.Apps.(*fakeApps)
	_, _ = fa.Create(t.Context(), apps.CreateInput{Name: "shop", RepoURL: "git@github.com:org/shop.git"})
	d.Specs = &apps.Specs{Apps: fa, Store: &memSpecStore{}, Now: time.Now}
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")
	return NewRouter(d), raw, d.Audit.(*fakeAudit)
}

func specBody(yaml string) string {
	b, _ := json.Marshal(map[string]string{"yaml": yaml})
	return string(b)
}

const goodSpec = "name: shop\nservices:\n  web:\n    kind: deployment\n    strategy: recreate\n    image: nginx\n"

func TestSpecAPI(t *testing.T) {
	h, raw, log := specRouter(t)

	if rec := bearer(h, http.MethodGet, APIPrefix+"/apps/1/spec", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("no spec yet: %d", rec.Code)
	}
	if rec := bearer(h, http.MethodPut, APIPrefix+"/apps/1/spec", raw, specBody(goodSpec)); rec.Code != http.StatusCreated {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	if rec := bearer(h, http.MethodPut, APIPrefix+"/apps/1/spec", raw, specBody(goodSpec)); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"created":false`) {
		t.Fatalf("unchanged save: %d %s", rec.Code, rec.Body)
	}
	changed := strings.Replace(goodSpec, "image: nginx", "image: nginx:1.27", 1)
	if rec := bearer(h, http.MethodPut, APIPrefix+"/apps/1/spec", raw, specBody(changed)); rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"version":2`) {
		t.Fatalf("second save: %d %s", rec.Code, rec.Body)
	}

	rec := bearer(h, http.MethodGet, APIPrefix+"/apps/1/spec/versions", raw, "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "nginx") {
		t.Fatalf("list must not include yaml: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps/1/spec/versions/1", raw, ""); !strings.Contains(rec.Body.String(), "image: nginx\\n") {
		t.Fatalf("get v1: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps/1/spec/diff?from=1&to=2", raw, ""); !strings.Contains(rec.Body.String(), "+    image: nginx:1.27") {
		t.Fatalf("diff: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps/1/spec/diff?from=x&to=2", raw, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad diff params: %d", rec.Code)
	}
	if got := strings.Count(strings.Join(log.actions(), " "), domain.AuditAppSpecUpdated); got != 2 {
		t.Fatalf("want 2 spec audit events, got %d", got)
	}
}

func TestSpecAPIProblems(t *testing.T) {
	h, raw, _ := specRouter(t)
	rec := bearer(h, http.MethodPut, APIPrefix+"/apps/1/spec", raw, specBody("services:\n  web:\n    kind: daemon\n    image: x\n    ports: [\"80:80\"]\n"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	var body struct {
		Error    string
		Problems []problemView
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Problems) == 0 || body.Problems[0].Line != 5 || !strings.Contains(body.Problems[0].Message, "host port bindings") {
		t.Fatalf("problems: %+v", body.Problems)
	}
}
