package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Backups is the backup repository.
type Backups struct {
	DB *sql.DB
}

const backupColumns = `id, env_id, app_name, env_name, service, volume, file, size_bytes, reason, created_at`

// CreateBackup records a finished backup.
func (r Backups) CreateBackup(ctx context.Context, b domain.Backup) (domain.Backup, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO backups (env_id, app_name, env_name, service, volume, file, size_bytes, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.EnvID, b.AppName, b.EnvName, b.Service, b.Volume, b.File, b.SizeBytes, string(b.Reason), formatTime(b.CreatedAt))
	if err != nil {
		return domain.Backup{}, err
	}
	if b.ID, err = res.LastInsertId(); err != nil {
		return domain.Backup{}, err
	}
	return b, nil
}

// GetBackup returns a backup or domain.ErrNotFound.
func (r Backups) GetBackup(ctx context.Context, id int64) (domain.Backup, error) {
	return scanBackup(r.DB.QueryRowContext(ctx, `SELECT `+backupColumns+` FROM backups WHERE id = ?`, id))
}

// ListBackups returns an environment's backups, newest first.
func (r Backups) ListBackups(ctx context.Context, envID int64) ([]domain.Backup, error) {
	return r.query(ctx, `SELECT `+backupColumns+` FROM backups WHERE env_id = ? ORDER BY id DESC`, envID)
}

// LatestBackup returns the newest backup of a volume or domain.ErrNotFound.
func (r Backups) LatestBackup(ctx context.Context, volume string) (domain.Backup, error) {
	return scanBackup(r.DB.QueryRowContext(ctx,
		`SELECT `+backupColumns+` FROM backups WHERE volume = ? ORDER BY id DESC LIMIT 1`, volume))
}

// BackupsBeyond returns a volume's backups older than the newest keep ones.
func (r Backups) BackupsBeyond(ctx context.Context, volume string, keep int) ([]domain.Backup, error) {
	return r.query(ctx, `SELECT `+backupColumns+` FROM backups WHERE volume = ? ORDER BY id DESC LIMIT -1 OFFSET ?`, volume, keep)
}

// DeleteBackup removes a backup record.
func (r Backups) DeleteBackup(ctx context.Context, id int64) error {
	return execOne(ctx, r.DB, `DELETE FROM backups WHERE id = ?`, id)
}

// BackupRetention returns how many backups per volume an environment keeps.
func (r Backups) BackupRetention(ctx context.Context, envID int64) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `SELECT backup_retention FROM environments WHERE id = ?`, envID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, domain.ErrNotFound
	}
	return n, err
}

// SetBackupRetention changes an environment's retention.
func (r Backups) SetBackupRetention(ctx context.Context, envID int64, keep int) error {
	return execOne(ctx, r.DB, `UPDATE environments SET backup_retention = ? WHERE id = ?`, keep, envID)
}

func (r Backups) query(ctx context.Context, q string, args ...any) ([]domain.Backup, error) {
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Backup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanBackup(row rowScanner) (domain.Backup, error) {
	var (
		b               domain.Backup
		envID           sql.NullInt64
		reason, created string
	)
	err := row.Scan(&b.ID, &envID, &b.AppName, &b.EnvName, &b.Service, &b.Volume, &b.File, &b.SizeBytes, &reason, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Backup{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Backup{}, err
	}
	if envID.Valid {
		b.EnvID = &envID.Int64
	}
	b.Reason = domain.BackupReason(reason)
	b.CreatedAt, err = parseTime(created)
	return b, err
}
