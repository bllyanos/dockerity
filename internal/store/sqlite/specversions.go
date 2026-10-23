package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
)

// SpecVersions stores immutable app spec versions. There is no update query.
type SpecVersions struct {
	DB *sql.DB
}

const specColumns = `app_id, version, yaml, created_at, created_by`

// InsertSpecVersion stores v; a taken version number returns domain.ErrConflict.
func (r SpecVersions) InsertSpecVersion(ctx context.Context, v domain.AppSpecVersion) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO app_spec_versions (`+specColumns+`) VALUES (?, ?, ?, ?, ?)`,
		v.AppID, v.Version, v.YAML, formatTime(v.CreatedAt), v.CreatedBy)
	if err != nil && isUniqueViolation(err) {
		return domain.ErrConflict
	}
	return err
}

// LatestSpecVersion returns the newest version or domain.ErrNotFound.
func (r SpecVersions) LatestSpecVersion(ctx context.Context, appID int64) (domain.AppSpecVersion, error) {
	return scanSpec(r.DB.QueryRowContext(ctx,
		`SELECT `+specColumns+` FROM app_spec_versions WHERE app_id = ? ORDER BY version DESC LIMIT 1`, appID))
}

// GetSpecVersion returns one version or domain.ErrNotFound.
func (r SpecVersions) GetSpecVersion(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error) {
	return scanSpec(r.DB.QueryRowContext(ctx,
		`SELECT `+specColumns+` FROM app_spec_versions WHERE app_id = ? AND version = ?`, appID, version))
}

// ListSpecVersions returns all versions without their YAML, newest first.
func (r SpecVersions) ListSpecVersions(ctx context.Context, appID int64) ([]domain.AppSpecVersion, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT app_id, version, '', created_at, created_by FROM app_spec_versions WHERE app_id = ? ORDER BY version DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AppSpecVersion
	for rows.Next() {
		v, err := scanSpec(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanSpec(row rowScanner) (domain.AppSpecVersion, error) {
	var (
		v         domain.AppSpecVersion
		created   string
		createdBy sql.NullInt64
	)
	err := row.Scan(&v.AppID, &v.Version, &v.YAML, &created, &createdBy)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AppSpecVersion{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.AppSpecVersion{}, err
	}
	if createdBy.Valid {
		v.CreatedBy = &createdBy.Int64
	}
	v.CreatedAt, err = parseTime(created)
	return v, err
}
