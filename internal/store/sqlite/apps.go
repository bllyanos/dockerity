package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Apps is the app repository.
type Apps struct {
	DB *sql.DB
}

const appColumns = `id, name, repo_url, default_branch, deploy_public_key, created_at, updated_at`

// ListApps returns all apps ordered by name.
func (r Apps) ListApps(ctx context.Context) ([]domain.App, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT `+appColumns+` FROM apps ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.App
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetApp returns the app or domain.ErrNotFound.
func (r Apps) GetApp(ctx context.Context, id int64) (domain.App, error) {
	return scanApp(r.DB.QueryRowContext(ctx, `SELECT `+appColumns+` FROM apps WHERE id = ?`, id))
}

// CreateApp inserts a. A taken name returns domain.ErrConflict.
func (r Apps) CreateApp(ctx context.Context, a domain.App) (domain.App, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO apps (name, repo_url, default_branch, deploy_public_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.Name, a.RepoURL, a.DefaultBranch, a.DeployPublicKey, formatTime(a.CreatedAt), formatTime(a.UpdatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.App{}, domain.ErrConflict
		}
		return domain.App{}, err
	}
	if a.ID, err = res.LastInsertId(); err != nil {
		return domain.App{}, err
	}
	return a, nil
}

// UpdateApp saves the mutable fields of a. The name never changes.
func (r Apps) UpdateApp(ctx context.Context, a domain.App) error {
	return execOne(ctx, r.DB, `
		UPDATE apps SET repo_url = ?, default_branch = ?, deploy_public_key = ?, updated_at = ? WHERE id = ?`,
		a.RepoURL, a.DefaultBranch, a.DeployPublicKey, formatTime(a.UpdatedAt), a.ID)
}

// DeleteApp removes the app or returns domain.ErrNotFound.
func (r Apps) DeleteApp(ctx context.Context, id int64) error {
	return execOne(ctx, r.DB, `DELETE FROM apps WHERE id = ?`, id)
}

func scanApp(row rowScanner) (domain.App, error) {
	var (
		a                domain.App
		created, updated string
	)
	err := row.Scan(&a.ID, &a.Name, &a.RepoURL, &a.DefaultBranch, &a.DeployPublicKey, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.App{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.App{}, err
	}
	if a.CreatedAt, err = parseTime(created); err != nil {
		return domain.App{}, err
	}
	if a.UpdatedAt, err = parseTime(updated); err != nil {
		return domain.App{}, err
	}
	return a, nil
}
