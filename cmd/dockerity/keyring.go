package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
)

// openKeyring loads the master key and fails fast if it is missing, invalid,
// or not the key that wrapped the stored data keys.
func openKeyring(ctx context.Context, logger *slog.Logger, keyFile string, db *sql.DB) (*secrets.Keyring, error) {
	master, err := crypto.ReadKeyFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	ring := secrets.NewKeyring(master, sqlite.DataKeys{DB: db}, time.Now)
	if err := ring.Verify(ctx); err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	logger.Info("master key loaded", "key_id", master.ID())
	return ring, nil
}
