package sqlite

import (
	"context"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// ListUsers returns all users ordered by username.
func (r Users) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateUser inserts u. A taken username returns domain.ErrConflict.
func (r Users) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, is_admin, disabled_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.IsAdmin, formatNullTime(u.DisabledAt), formatTime(u.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrConflict
		}
		return domain.User{}, err
	}
	if u.ID, err = res.LastInsertId(); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// SetUserDisabled disables the user at the given time, or enables them when at is nil.
func (r Users) SetUserDisabled(ctx context.Context, id int64, at *time.Time) error {
	return execOne(ctx, r.DB, `UPDATE users SET disabled_at = ? WHERE id = ?`, formatNullTime(at), id)
}

// SetUserPassword replaces the password hash.
func (r Users) SetUserPassword(ctx context.Context, id int64, hash string) error {
	return execOne(ctx, r.DB, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
}
