package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func bearer(h http.Handler, method, path, raw, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestTokenLifecycle(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)
	sessionRaw, _, _ := d.Sessions.(*fakeSessions).Start(t.Context(), alice)
	cookie := &http.Cookie{Name: SessionCookie, Value: sessionRaw}

	req := httptest.NewRequest(http.MethodPost, APIPrefix+"/tokens", strings.NewReader(`{"name":"ci"}`))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created createTokenResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Secret == "" || created.Token.Name != "ci" || strings.Contains(rec.Body.String(), "digest") {
		t.Fatalf("unexpected create response %s", rec.Body)
	}

	// The token authenticates API calls, without setting a session cookie.
	rec = bearer(h, http.MethodGet, APIPrefix+"/auth/me", created.Secret, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"alice"`) {
		t.Fatalf("me with token: %d %s", rec.Code, rec.Body)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("bearer requests must not set cookies")
	}
	bearer(h, http.MethodGet, APIPrefix+"/auth/me", created.Secret, "")

	rec = bearer(h, http.MethodGet, APIPrefix+"/tokens", created.Secret, "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), created.Secret) {
		t.Fatalf("list must not return the secret: %s", rec.Body)
	}

	if rec = bearer(h, http.MethodDelete, APIPrefix+"/tokens/1", created.Secret, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/auth/me", created.Secret, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token still works: %d", rec.Code)
	}

	want := []string{domain.AuditTokenCreated, domain.AuditTokenUsed, domain.AuditTokenRevoked}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("audit %v, want %v (token use recorded once)", got, want)
	}
	used := log.events[1]
	if used.ActorTokenID == nil || *used.ActorTokenID != 1 || used.ActorUserID == nil {
		t.Fatalf("token.used actor: %+v", used)
	}
}

func TestBadBearerDoesNotFallBackToCookie(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	sessionRaw, _, _ := d.Sessions.(*fakeSessions).Start(t.Context(), alice)
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookie, Value: sessionRaw})
	req.Header.Set("Authorization", "Bearer nope")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestRevokeValidation(t *testing.T) {
	d := fullDeps()
	h := NewRouter(d)
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "ci")
	if rec := bearer(h, http.MethodDelete, APIPrefix+"/tokens/abc", raw, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("non-numeric id: %d", rec.Code)
	}
	if rec := bearer(h, http.MethodDelete, APIPrefix+"/tokens/99", raw, ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown id: %d", rec.Code)
	}
}
