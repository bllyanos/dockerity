package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/gitx"
)

// KnownHosts stores pinned SSH host keys.
type KnownHosts struct {
	DB *sql.DB
}

// GetHostKey returns the pinned key for host or domain.ErrNotFound.
func (r KnownHosts) GetHostKey(ctx context.Context, host string) (gitx.KnownHost, error) {
	k := gitx.KnownHost{Host: host}
	var added string
	err := r.DB.QueryRowContext(ctx,
		`SELECT key_type, key, fingerprint, added_at FROM known_hosts WHERE host = ?`, host).
		Scan(&k.KeyType, &k.Key, &k.Fingerprint, &added)
	if errors.Is(err, sql.ErrNoRows) {
		return gitx.KnownHost{}, domain.ErrNotFound
	}
	if err != nil {
		return gitx.KnownHost{}, err
	}
	k.AddedAt, err = parseTime(added)
	return k, err
}

// AddHostKey pins k unless the host already has a pinned key.
func (r KnownHosts) AddHostKey(ctx context.Context, k gitx.KnownHost) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO known_hosts (host, key_type, key, fingerprint, added_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (host) DO NOTHING`,
		k.Host, k.KeyType, k.Key, k.Fingerprint, formatTime(k.AddedAt))
	return err
}
