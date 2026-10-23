package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// DataKeys stores wrapped data keys.
type DataKeys struct {
	DB *sql.DB
}

// GetDataKey returns the scope's wrapped key or domain.ErrNotFound.
func (r DataKeys) GetDataKey(ctx context.Context, scope secrets.Scope) (secrets.WrappedKey, error) {
	var (
		k       = secrets.WrappedKey{Scope: scope}
		created string
	)
	err := r.DB.QueryRowContext(ctx,
		`SELECT wrapped_key, nonce, master_key_id, created_at FROM data_keys WHERE scope = ?`, string(scope)).
		Scan(&k.Sealed.Ciphertext, &k.Sealed.Nonce, &k.MasterKeyID, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return secrets.WrappedKey{}, domain.ErrNotFound
	}
	if err != nil {
		return secrets.WrappedKey{}, err
	}
	k.CreatedAt, err = parseTime(created)
	return k, err
}

// InsertDataKey stores k unless the scope already has a key.
func (r DataKeys) InsertDataKey(ctx context.Context, k secrets.WrappedKey) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO data_keys (scope, wrapped_key, nonce, master_key_id, created_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT (scope) DO NOTHING`,
		string(k.Scope), k.Sealed.Ciphertext, k.Sealed.Nonce, k.MasterKeyID, formatTime(k.CreatedAt))
	return err
}

// CountDataKeysNotWrappedBy counts keys wrapped by a master key other than masterKeyID.
func (r DataKeys) CountDataKeysNotWrappedBy(ctx context.Context, masterKeyID string) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM data_keys WHERE master_key_id <> ?`, masterKeyID).Scan(&n)
	return n, err
}
