package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Environments is the environment repository.
type Environments struct {
	DB *sql.DB
}

const envSelect = `SELECT e.id, e.app_id, a.name, e.name, e.type, e.domain, e.ttl_seconds, e.expires_at, e.resource_overrides, e.created_at, e.updated_at
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

// ListAllEnvironments returns every environment of every app.
func (r Environments) ListAllEnvironments(ctx context.Context) ([]domain.Environment, error) {
	rows, err := r.DB.QueryContext(ctx, envSelect+` ORDER BY e.id`)
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
		INSERT INTO environments (app_id, name, type, domain, ttl_seconds, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.AppID, e.Name, string(e.Type), e.Domain, int64(e.TTL/time.Second), formatNullTime(e.ExpiresAt),
		formatTime(e.CreatedAt), formatTime(e.UpdatedAt))
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

// SetResources replaces an environment's resource overrides.
func (r Environments) SetResources(ctx context.Context, id int64, o domain.ResourceOverrides) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return execOne(ctx, r.DB, `UPDATE environments SET resource_overrides = ? WHERE id = ?`, string(data), id)
}

// CountEphemeral counts an app's ephemeral environments.
func (r Environments) CountEphemeral(ctx context.Context, appID int64) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM environments WHERE app_id = ? AND type = ?`, appID, string(domain.EnvEphemeral)).Scan(&n)
	return n, err
}

// SetExpiry moves an environment's expiry.
func (r Environments) SetExpiry(ctx context.Context, id int64, expires time.Time) error {
	return execOne(ctx, r.DB, `UPDATE environments SET expires_at = ? WHERE id = ? AND expires_at IS NOT NULL`, formatTime(expires), id)
}

// ListExpired returns environments whose expiry is at or before now.
func (r Environments) ListExpired(ctx context.Context, now time.Time) ([]domain.Environment, error) {
	rows, err := r.DB.QueryContext(ctx, envSelect+` WHERE e.expires_at IS NOT NULL AND e.expires_at <= ? ORDER BY e.expires_at`, formatTime(now))
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

func scanEnv(row rowScanner) (domain.Environment, error) {
	var (
		e                     domain.Environment
		typ, created, updated string
		ttl                   int64
		expires               sql.NullString
		overrides             string
	)
	err := row.Scan(&e.ID, &e.AppID, &e.AppName, &e.Name, &typ, &e.Domain, &ttl, &expires, &overrides, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Environment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Environment{}, err
	}
	e.Type = domain.EnvType(typ)
	e.TTL = time.Duration(ttl) * time.Second
	if err := json.Unmarshal([]byte(overrides), &e.Resources); err != nil {
		return domain.Environment{}, err
	}
	if e.ExpiresAt, err = parseNullTime(expires); err != nil {
		return domain.Environment{}, err
	}
	if e.CreatedAt, err = parseTime(created); err != nil {
		return domain.Environment{}, err
	}
	e.UpdatedAt, err = parseTime(updated)
	return e, err
}
