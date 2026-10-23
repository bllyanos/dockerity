package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Environments is the environment repository.
type Environments struct {
	DB *sql.DB
}

const envSelect = `SELECT e.id, e.app_id, a.name, e.name, e.type, e.domain, e.created_at, e.updated_at
	FROM environments e JOIN apps a ON a.id = e.app_id`

// ListEnvironments returns the app's environments ordered by name.
func (r Environments) ListEnvironments(ctx context.Context, appID int64) ([]domain.Environment, error) {
	rows, err := r.DB.QueryContext(ctx, envSelect+` WHERE e.app_id = ? ORDER BY e.name`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Environment
	for rows.Next() {
		e, err := scanEnv(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetEnvironment returns the environment or domain.ErrNotFound.
func (r Environments) GetEnvironment(ctx context.Context, id int64) (domain.Environment, error) {
	return scanEnv(r.DB.QueryRowContext(ctx, envSelect+` WHERE e.id = ?`, id))
}

// CreateEnvironment inserts e. A taken name or domain returns domain.ErrConflict.
func (r Environments) CreateEnvironment(ctx context.Context, e domain.Environment) (domain.Environment, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO environments (app_id, name, type, domain, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		e.AppID, e.Name, string(e.Type), e.Domain, formatTime(e.CreatedAt), formatTime(e.UpdatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Environment{}, domain.ErrConflict
		}
		return domain.Environment{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Environment{}, err
	}
	return r.GetEnvironment(ctx, id)
}

// UpdateEnvironment saves the domain. A taken domain returns domain.ErrConflict.
func (r Environments) UpdateEnvironment(ctx context.Context, e domain.Environment) error {
	err := execOne(ctx, r.DB, `UPDATE environments SET domain = ?, updated_at = ? WHERE id = ?`,
		e.Domain, formatTime(e.UpdatedAt), e.ID)
	if err != nil && isUniqueViolation(err) {
		return domain.ErrConflict
	}
	return err
}

// DeleteEnvironment removes the environment or returns domain.ErrNotFound.
func (r Environments) DeleteEnvironment(ctx context.Context, id int64) error {
	return execOne(ctx, r.DB, `DELETE FROM environments WHERE id = ?`, id)
}

func scanEnv(row rowScanner) (domain.Environment, error) {
	var (
		e                     domain.Environment
		typ, created, updated string
	)
	err := row.Scan(&e.ID, &e.AppID, &e.AppName, &e.Name, &typ, &e.Domain, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Environment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Environment{}, err
	}
	e.Type = domain.EnvType(typ)
	if e.CreatedAt, err = parseTime(created); err != nil {
		return domain.Environment{}, err
	}
	e.UpdatedAt, err = parseTime(updated)
	return e, err
}
