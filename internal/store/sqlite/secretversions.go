package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// SecretVersions stores encrypted secret set versions.
type SecretVersions struct {
	DB *sql.DB
}

const secretVersionColumns = `scope, version, key_names, ciphertext, nonce, created_at, created_by`

// LatestSecretVersion returns the newest version or domain.ErrNotFound.
func (r SecretVersions) LatestSecretVersion(ctx context.Context, scope secrets.Scope) (secrets.EncryptedVersion, error) {
	return scanSecretVersion(r.DB.QueryRowContext(ctx,
		`SELECT `+secretVersionColumns+` FROM secret_set_versions WHERE scope = ? ORDER BY version DESC LIMIT 1`, string(scope)))
}

// GetSecretVersion returns one version or domain.ErrNotFound.
func (r SecretVersions) GetSecretVersion(ctx context.Context, scope secrets.Scope, version int) (secrets.EncryptedVersion, error) {
	return scanSecretVersion(r.DB.QueryRowContext(ctx,
		`SELECT `+secretVersionColumns+` FROM secret_set_versions WHERE scope = ? AND version = ?`, string(scope), version))
}

// ListSecretVersions returns version metadata, newest first.
func (r SecretVersions) ListSecretVersions(ctx context.Context, scope secrets.Scope) ([]secrets.Version, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT `+secretVersionColumns+` FROM secret_set_versions WHERE scope = ? ORDER BY version DESC`, string(scope))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []secrets.Version
	for rows.Next() {
		ev, err := scanSecretVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev.Version)
	}
	return out, rows.Err()
}

// InsertSecretVersion stores v; a taken version number returns domain.ErrConflict.
func (r SecretVersions) InsertSecretVersion(ctx context.Context, v secrets.EncryptedVersion) error {
	names, err := json.Marshal(v.Keys)
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO secret_set_versions (`+secretVersionColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(v.Scope), v.Version.Version, string(names), v.Sealed.Ciphertext, v.Sealed.Nonce, formatTime(v.CreatedAt), v.CreatedBy)
	if err != nil && isUniqueViolation(err) {
		return domain.ErrConflict
	}
	return err
}

func scanSecretVersion(row rowScanner) (secrets.EncryptedVersion, error) {
	var (
		ev               secrets.EncryptedVersion
		scope, names, at string
		createdBy        sql.NullInt64
	)
	err := row.Scan(&scope, &ev.Version.Version, &names, &ev.Sealed.Ciphertext, &ev.Sealed.Nonce, &at, &createdBy)
	if errors.Is(err, sql.ErrNoRows) {
		return secrets.EncryptedVersion{}, domain.ErrNotFound
	}
	if err != nil {
		return secrets.EncryptedVersion{}, err
	}
	ev.Scope = secrets.Scope(scope)
	if err := json.Unmarshal([]byte(names), &ev.Keys); err != nil {
		return secrets.EncryptedVersion{}, err
	}
	if ev.CreatedAt, err = parseTime(at); err != nil {
		return secrets.EncryptedVersion{}, err
	}
	if createdBy.Valid {
		ev.CreatedBy = &createdBy.Int64
	}
	return ev, nil
}
