package sqlite

import (
	"context"
	"database/sql"
	"time"
)

// Settings stores instance-wide key/value settings.
type Settings struct {
	DB *sql.DB
}

// AllSettings returns every stored setting.
func (r Settings) AllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// SaveSettings upserts all values in one transaction.
func (r Settings) SaveSettings(ctx context.Context, values map[string]string, at time.Time) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range values {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			k, v, formatTime(at)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
