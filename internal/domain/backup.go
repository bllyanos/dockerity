package domain

import "time"

// Audit actions for backups.
const (
	AuditBackupCreated  = "backup.created"
	AuditBackupRestored = "backup.restored"
)

// BackupReason tells why a backup was taken.
type BackupReason string

// Backup reasons.
const (
	BackupManual      BackupReason = "manual"
	BackupImageChange BackupReason = "image-change"
	BackupEnvDelete   BackupReason = "env-delete"
)

// DefaultBackupRetention is how many backups per volume are kept by default.
const DefaultBackupRetention = 5

// Backup is an archive of one volume. It outlives its environment: EnvID is
// nil once the environment is deleted, and AppName/EnvName keep the origin.
type Backup struct {
	ID        int64
	EnvID     *int64
	AppName   string
	EnvName   string
	Service   string
	Volume    string // Docker volume name
	File      string // archive name inside the backups volume
	SizeBytes int64
	Reason    BackupReason
	CreatedAt time.Time
}
