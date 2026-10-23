package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestAuthEventsAreAudited(t *testing.T) {
	d := fullDeps()
	log := d.Audit.(*fakeAudit)
	h := NewRouter(d)

	do(h, http.MethodPost, APIPrefix+"/setup", `{"username":"alice","password":"long enough pw"}`)
	do(h, http.MethodPost, APIPrefix+"/auth/login", `{"username":"alice","password":"wrong"}`)
	rec := do(h, http.MethodPost, APIPrefix+"/auth/login", `{"username":"alice","password":"pw"}`)
	withCookie(h, http.MethodPost, APIPrefix+"/auth/logout", cookieFrom(t, rec))

	want := []string{domain.AuditSetupCompleted, domain.AuditLoginFailed, domain.AuditLogin, domain.AuditLogout}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("actions %v, want %v", got, want)
	}
	failed := log.events[1]
	if failed.ActorUserID != nil || failed.Metadata["username"] != "alice" || failed.RemoteAddr == "" {
		t.Fatalf("failed login event %+v", failed)
	}
	for _, e := range []domain.AuditEvent{log.events[2], log.events[3]} {
		if e.ActorUserID == nil || *e.ActorUserID != alice.ID || e.TargetID != "alice" {
			t.Fatalf("event %s missing actor: %+v", e.Action, e)
		}
	}
}

func TestAuditListAdminOnly(t *testing.T) {
	d := fullDeps()
	sessions := d.Sessions.(*fakeSessions)
	log := d.Audit.(*fakeAudit)
	for range 3 {
		_ = log.Record(t.Context(), domain.AuditEvent{Action: domain.AuditLogin})
	}
	h := NewRouter(d)

	adminRaw, _, _ := sessions.Start(t.Context(), alice)
	rec := withCookie(h, http.MethodGet, APIPrefix+"/audit?limit=2", &http.Cookie{Name: SessionCookie, Value: adminRaw})
	if rec.Code != http.StatusOK {
		t.Fatalf("admin: %d %s", rec.Code, rec.Body)
	}
	var page struct {
		Events     []auditEventView
		NextBefore *int64
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 2 || page.Events[0].ID != 3 || page.NextBefore == nil || *page.NextBefore != 2 {
		t.Fatalf("unexpected page %+v", page)
	}

	bob := domain.User{ID: 2, Username: "bob"}
	userRaw, _, _ := sessions.Start(t.Context(), bob)
	if rec = withCookie(h, http.MethodGet, APIPrefix+"/audit", &http.Cookie{Name: SessionCookie, Value: userRaw}); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin: got %d", rec.Code)
	}

	for _, q := range []string{"?limit=0", "?limit=999", "?before=x"} {
		if rec = withCookie(h, http.MethodGet, APIPrefix+"/audit"+q, &http.Cookie{Name: SessionCookie, Value: adminRaw}); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d", q, rec.Code)
		}
	}
}
