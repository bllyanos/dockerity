// Package sqlite is the SQLite-backed persistence layer.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// FileName is the database file name inside the data dir.
const FileName = "dockerity.db"

// pragmas are applied to every new connection.
var pragmas = []string{
	"journal_mode(WAL)",
	"foreign_keys(1)",
	"busy_timeout(5000)",
	"synchronous(NORMAL)",
}

// Open creates dataDir if needed and opens the database inside it.
func Open(ctx context.Context, dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return OpenFile(ctx, filepath.Join(dataDir, FileName))
}

// OpenFile opens the database at path with Dockerity's pragmas.
func OpenFile(ctx context.Context, path string) (*sql.DB, error) {
	q := url.Values{}
	for _, p := range pragmas {
		q.Add("_pragma", p)
	}
	// Writers take the lock at BEGIN, so concurrent writers wait on busy_timeout
	// instead of failing with SQLITE_BUSY when upgrading a read lock.
	q.Set("_txlock", "immediate")

	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return db, nil
}
