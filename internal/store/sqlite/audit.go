package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/bllyanos/dockerity/internal/audit"
	"github.com/bllyanos/dockerity/internal/domain"
)

// AuditLog is the append-only audit event store.
type AuditLog struct {
	DB *sql.DB
}

// Record appends e. Metadata is sanitized again here as a last line of defense.
func (r AuditLog) Record(ctx context.Context, e domain.AuditEvent) error {
	meta, err := json.Marshal(audit.Sanitize(e.Metadata))
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO audit_events (at, actor_user_id, actor_token_id, action, target_type, target_id, metadata, remote_addr)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		formatTime(e.At), e.ActorUserID, e.ActorTokenID, e.Action, e.TargetType, e.TargetID, string(meta), e.RemoteAddr)
	return err
}

// ListAudit returns up to limit events older than before (by id), newest
// first. before <= 0 starts from the newest event.
func (r AuditLog) ListAudit(ctx context.Context, before int64, limit int) ([]domain.AuditEvent, error) {
	query := `
		SELECT e.id, e.at, e.actor_user_id, e.actor_token_id, COALESCE(u.username, ''),
		       e.action, e.target_type, e.target_id, e.metadata, e.remote_addr
		FROM audit_events e LEFT JOIN users u ON u.id = e.actor_user_id`
	args := []any{}
	if before > 0 {
		query += ` WHERE e.id < ?`
		args = append(args, before)
	}
	query += ` ORDER BY e.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AuditEvent
	for rows.Next() {
		var (
			e        domain.AuditEvent
			at, meta string
			userID   sql.NullInt64
			tokenID  sql.NullInt64
		)
		if err := rows.Scan(&e.ID, &at, &userID, &tokenID, &e.ActorUsername,
			&e.Action, &e.TargetType, &e.TargetID, &meta, &e.RemoteAddr); err != nil {
			return nil, err
		}
		if e.At, err = parseTime(at); err != nil {
			return nil, err
		}
		if userID.Valid {
			e.ActorUserID = &userID.Int64
		}
		if tokenID.Valid {
			e.ActorTokenID = &tokenID.Int64
		}
		if err := json.Unmarshal([]byte(meta), &e.Metadata); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
