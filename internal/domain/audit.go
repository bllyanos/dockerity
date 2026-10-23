package domain

import "time"

// Audit actions. Later epics add their own (deploys, secrets, environments).
const (
	AuditSetupCompleted = "setup.completed"
	AuditLogin          = "auth.login"
	AuditLoginFailed    = "auth.login_failed"
	AuditLogout         = "auth.logout"
)

// AuditEvent is one entry in the append-only audit log.
type AuditEvent struct {
	ID int64
	At time.Time
	// ActorUserID is nil for anonymous actions such as a failed login.
	ActorUserID *int64
	// ActorTokenID is set when the action was made with an API token.
	ActorTokenID *int64
	// ActorUsername is filled when reading the log; it is not stored.
	ActorUsername string
	Action        string
	TargetType    string
	TargetID      string
	// Metadata holds names and ids only, never secret values.
	Metadata   map[string]string
	RemoteAddr string
}
