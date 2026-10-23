package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Users is the user repository.
type Users struct {
	DB *sql.DB
}

const userColumns = `id, username, password_hash, is_admin, disabled_at, created_at`

// CountUsers returns the number of users.
func (r Users) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// CreateFirstAdmin inserts u only if the table is empty, in one statement so
// concurrent setup requests cannot both succeed.
func (r Users) CreateFirstAdmin(ctx context.Context, u domain.User) (domain.User, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, is_admin, disabled_at, created_at)
		SELECT ?, ?, ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM users)`,
		u.Username, u.PasswordHash, u.IsAdmin, formatNullTime(u.DisabledAt), formatTime(u.CreatedAt))
	if err != nil {
		return domain.User{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return domain.User{}, err
	} else if n == 0 {
		return domain.User{}, domain.ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}
	u.ID = id
	return u, nil
}

// GetUserByID returns the user or domain.ErrNotFound.
func (r Users) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(r.DB.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id))
}

// GetUserByUsername returns the user or domain.ErrNotFound.
func (r Users) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	return scanUser(r.DB.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE username = ?`, username))
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (domain.User, error) {
	var (
		u        domain.User
		disabled sql.NullString
		created  string
	)
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &disabled, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	if u.DisabledAt, err = parseNullTime(disabled); err != nil {
		return domain.User{}, err
	}
	if u.CreatedAt, err = parseTime(created); err != nil {
		return domain.User{}, err
	}
	return u, nil
}
