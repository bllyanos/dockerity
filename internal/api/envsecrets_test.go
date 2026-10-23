package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/envsecrets"
)

func TestEnvSecretsAreWriteOnly(t *testing.T) {
	d := fullDeps()
	var logs bytes.Buffer
	d.Logger = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	envs := d.Environments.(*fakeEnvs)
	_, _ = envs.CreatePermanent(t.Context(), 1, "staging", "")
	d.EnvSecrets = &envsecrets.Service{Envs: envs, Secrets: newMemSecretStore()}
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")

	rec := bearer(h, http.MethodGet, APIPrefix+"/environments/1/secrets", raw, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"version":0`) {
		t.Fatalf("empty: %d %s", rec.Code, rec.Body)
	}

	const value = "s3cr3t-value-123"
	var bodies []string
	rec = bearer(h, http.MethodPatch, APIPrefix+"/environments/1/secrets", raw, `{"set":{"DB_PASSWORD":"`+value+`","API_KEY":"k"}}`)
	bodies = append(bodies, rec.Body.String())
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"keys":["API_KEY","DB_PASSWORD"]`) {
		t.Fatalf("set: %d %s", rec.Code, rec.Body)
	}
	rec = bearer(h, http.MethodPatch, APIPrefix+"/environments/1/secrets", raw, `{"unset":["API_KEY"]}`)
	bodies = append(bodies, rec.Body.String())
	rec = bearer(h, http.MethodPatch, APIPrefix+"/environments/1/secrets", raw, `{"set":{"bad key":"`+value+`"}}`)
	bodies = append(bodies, rec.Body.String())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid key: %d", rec.Code)
	}
	for _, path := range []string{"/environments/1/secrets", "/environments/1/secrets/versions"} {
		rec = bearer(h, http.MethodGet, APIPrefix+path, raw, "")
		bodies = append(bodies, rec.Body.String())
	}
	if !strings.Contains(bodies[len(bodies)-1], `"version":2`) {
		t.Fatalf("versions: %s", bodies[len(bodies)-1])
	}

	for _, b := range bodies {
		if strings.Contains(b, value) {
			t.Fatalf("secret value in response: %s", b)
		}
	}
	if strings.Contains(logs.String(), value) {
		t.Fatal("secret value in logs")
	}
	for _, e := range log.events {
		if e.Action == domain.AuditSecretsUpdated {
			for _, v := range e.Metadata {
				if strings.Contains(v, value) {
					t.Fatalf("secret value in audit: %v", e.Metadata)
				}
			}
		}
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/environments/9/secrets", raw, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing env: %d", rec.Code)
	}
}
