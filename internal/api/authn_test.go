package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/auth/ratelimit"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/envsecrets"
)

func fullDeps() Deps {
	return Deps{Logger: discardLogger, Setup: &fakeSetup{}, Sessions: newFakeSessions(), Tokens: newFakeTokens(), Audit: &fakeAudit{},
		Users: &fakeUserAdmin{users: []domain.User{alice}}, Apps: &fakeApps{}, ConnectionTester: fakeTester{}, Builds: &fakeBuilds{}, Environments: &fakeEnvs{}, Deployments: &fakeDeployments{}, EnvContainers: fakeEnvContainers{},
		EnvSecrets: &envsecrets.Service{Envs: &fakeEnvs{}, Secrets: newMemSecretStore()},
		Specs:      &apps.Specs{Apps: &fakeApps{}, Store: &memSpecStore{}, Now: time.Now}}
}

// concretePath turns a mux pattern like "DELETE /tokens/{id}" into a method and path.
func concretePath(pattern string) (string, string) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		method, path = http.MethodGet, pattern
	}
	for strings.Contains(path, "{") {
		start := strings.Index(path, "{")
		end := strings.Index(path, "}")
		path = path[:start] + "1" + path[end+1:]
	}
	return method, APIPrefix + path
}

// TestEveryProtectedRouteRequiresAuth walks every registered API route, so a
// new route is protected unless it is explicitly registered as public.
func TestEveryProtectedRouteRequiresAuth(t *testing.T) {
	d := fullDeps()
	rt := apiRoutes(d)
	h := NewRouter(d)

	allowedPublic := map[string]bool{"GET /setup": true, "POST /setup": true, "POST /auth/login": true}
	for pattern := range rt.public {
		if !allowedPublic[pattern] {
			t.Errorf("unexpected public route %q", pattern)
		}
	}
	for _, pattern := range rt.patterns {
		if rt.public[pattern] {
			continue
		}
		method, path := concretePath(pattern)
		rec := do(h, method, path, "{}")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without auth: got %d", method, path, rec.Code)
		}
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: SessionCookie, Value: "dkts_forged"})
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with forged cookie: got %d", method, path, rec.Code)
		}
		req = httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer dkta_forged")
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with forged bearer: got %d", method, path, rec.Code)
		}
	}
}

func TestProtectedRoutesDeniedWithoutSessionService(t *testing.T) {
	h := NewRouter(Deps{Logger: discardLogger})
	if rec := do(h, http.MethodGet, APIPrefix+"/auth/me", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func cookieFrom(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie {
			return c
		}
	}
	t.Fatalf("no %s cookie in response", SessionCookie)
	return nil
}

func withCookie(h http.Handler, method, path string, c *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLoginMeLogout(t *testing.T) {
	h := NewRouter(fullDeps())

	rec := do(h, http.MethodPost, APIPrefix+"/auth/login", `{"username":"alice","password":"pw"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	c := cookieFrom(t, rec)
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure {
		t.Fatalf("unexpected cookie attributes %+v", c)
	}

	rec = withCookie(h, http.MethodGet, APIPrefix+"/auth/me", c)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"username":"alice"`) {
		t.Fatalf("me: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "secret-hash") {
		t.Fatal("password hash leaked")
	}

	if rec = withCookie(h, http.MethodPost, APIPrefix+"/auth/logout", c); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rec.Code)
	}
	if cleared := cookieFrom(t, rec); cleared.MaxAge >= 0 {
		t.Fatalf("cookie not cleared: %+v", cleared)
	}
	if rec = withCookie(h, http.MethodGet, APIPrefix+"/auth/me", c); rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout: %d", rec.Code)
	}
}

func TestSecureCookieBehindTLSProxy(t *testing.T) {
	h := NewRouter(fullDeps())
	req := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/login", strings.NewReader(`{"username":"alice","password":"pw"}`))
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !cookieFrom(t, rec).Secure {
		t.Fatal("cookie must be Secure over HTTPS")
	}
}

func TestLoginRateLimited(t *testing.T) {
	d := fullDeps()
	d.LoginLimiter = ratelimit.New(2, time.Minute)
	h := NewRouter(d)
	bad := `{"username":"alice","password":"nope"}`

	for range 2 {
		if rec := do(h, http.MethodPost, APIPrefix+"/auth/login", bad); rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d", rec.Code)
		}
	}
	rec := do(h, http.MethodPost, APIPrefix+"/auth/login", `{"username":"alice","password":"pw"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429 even for the right password", rec.Code)
	}
}

func TestSetupLogsAdminIn(t *testing.T) {
	h := NewRouter(fullDeps())
	rec := do(h, http.MethodPost, APIPrefix+"/setup", `{"username":"alice","password":"long enough pw"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: %d %s", rec.Code, rec.Body)
	}
	if rec = withCookie(h, http.MethodGet, APIPrefix+"/auth/me", cookieFrom(t, rec)); rec.Code != http.StatusOK {
		t.Fatalf("me after setup: %d", rec.Code)
	}
}

func TestCrossOriginMutationRejected(t *testing.T) {
	h := NewRouter(fullDeps())
	req := httptest.NewRequest(http.MethodPost, APIPrefix+"/auth/login", strings.NewReader(`{"username":"alice","password":"pw"}`))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		remote string
		xff    []string
		want   string
	}{
		{"10.0.0.2:5555", nil, "10.0.0.2"},
		{"10.0.0.2:5555", []string{"1.1.1.1"}, "1.1.1.1"},
		{"10.0.0.2:5555", []string{"6.6.6.6, 1.1.1.1"}, "1.1.1.1"},
		{"10.0.0.2:5555", []string{"6.6.6.6", "1.1.1.1"}, "1.1.1.1"},
		{"10.0.0.2:5555", []string{" "}, "10.0.0.2"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = c.remote
		for _, v := range c.xff {
			req.Header.Add("X-Forwarded-For", v)
		}
		if got := clientIP(req); got != c.want {
			t.Errorf("%v: got %q, want %q", c, got, c.want)
		}
	}
}
