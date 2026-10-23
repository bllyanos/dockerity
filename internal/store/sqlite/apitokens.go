package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// APITokens is the API token repository.
type APITokens struct {
	DB *sql.DB
}

const apiTokenColumns = `id, user_id, name, token_digest, created_at, last_used_at, revoked_at`

// CreateAPIToken stores t and returns it with its ID.
func (r APITokens) CreateAPIToken(ctx context.Context, t domain.APIToken) (domain.APIToken, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO api_tokens (user_id, name, token_digest, created_at, last_used_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		t.UserID, t.Name, t.TokenDigest, formatTime(t.CreatedAt), formatNullTime(t.LastUsedAt), formatNullTime(t.RevokedAt))
	if err != nil {
		return domain.APIToken{}, err
	}
	if t.ID, err = res.LastInsertId(); err != nil {
		return domain.APIToken{}, err
	}
	return t, nil
}

// ListAPITokens returns the user's tokens, newest first, including revoked ones.
func (r APITokens) ListAPITokens(ctx context.Context, userID int64) ([]domain.APIToken, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT `+apiTokenColumns+` FROM api_tokens WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.APIToken
	for rows.Next() {
		t, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetAPITokenByDigest returns the token or domain.ErrNotFound.
func (r APITokens) GetAPITokenByDigest(ctx context.Context, digest string) (domain.APIToken, error) {
	return scanAPIToken(r.DB.QueryRowContext(ctx,
		`SELECT `+apiTokenColumns+` FROM api_tokens WHERE token_digest = ?`, digest))
}

// RevokeAPIToken marks the user's token revoked. It returns domain.ErrNotFound
// when the token does not exist, belongs to someone else, or is already revoked.
func (r APITokens) RevokeAPIToken(ctx context.Context, userID, id int64, at time.Time) (domain.APIToken, error) {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE api_tokens SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
		formatTime(at), id, userID)
	if err != nil {
		return domain.APIToken{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return domain.APIToken{}, err
	} else if n == 0 {
		return domain.APIToken{}, domain.ErrNotFound
	}
	return scanAPIToken(r.DB.QueryRowContext(ctx, `SELECT `+apiTokenColumns+` FROM api_tokens WHERE id = ?`, id))
}

// TouchAPIToken records the last use time.
func (r APITokens) TouchAPIToken(ctx context.Context, id int64, at time.Time) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, formatTime(at), id)
	return err
}

// RevokeUserAPITokens revokes every active token of a user.
func (r APITokens) RevokeUserAPITokens(ctx context.Context, userID int64, at time.Time) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE api_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, formatTime(at), userID)
	return err
}

func scanAPIToken(row rowScanner) (domain.APIToken, error) {
	var (
		t                 domain.APIToken
		created           string
		lastUsed, revoked sql.NullString
	)
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenDigest, &created, &lastUsed, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.APIToken{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.APIToken{}, err
	}
	if t.CreatedAt, err = parseTime(created); err != nil {
		return domain.APIToken{}, err
	}
	if t.LastUsedAt, err = parseNullTime(lastUsed); err != nil {
		return domain.APIToken{}, err
	}
	if t.RevokedAt, err = parseNullTime(revoked); err != nil {
		return domain.APIToken{}, err
	}
	return t, nil
}
