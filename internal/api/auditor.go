package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bllyanos/dockerity/internal/audit"
	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
)

// auditor records audit events for HTTP requests. A nil recorder is a no-op.
type auditor struct {
	rec    audit.Recorder
	logger *slog.Logger
}

// record fills time, actor (from the request principal unless already set)
// and remote address, then appends e. Failures are logged, not returned, so
// an audit outage does not block the action itself.
func (a auditor) record(r *http.Request, e domain.AuditEvent) {
	if a.rec == nil {
		return
	}
	e.At = time.Now().UTC()
	if p, ok := auth.PrincipalFrom(r.Context()); ok && e.ActorUserID == nil {
		id := p.User.ID
		e.ActorUserID = &id
		if p.APITokenID != 0 {
			tid := p.APITokenID
			e.ActorTokenID = &tid
		}
	}
	e.RemoteAddr = clientIP(r)
	if err := a.rec.Record(r.Context(), e); err != nil {
		a.logger.ErrorContext(r.Context(), "audit record failed", "action", e.Action, "error", err)
	}
}

func userTarget(u domain.User) (string, string) {
	return "user", u.Username
}
