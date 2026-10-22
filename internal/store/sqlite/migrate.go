package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed migrations/*.sql
var embedded embed.FS

// Migrations returns Dockerity's embedded schema migrations.
func Migrations() ([]Migration, error) {
	sub, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return nil, err
	}
	return LoadMigrations(sub)
}

const createMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version    INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
)`

// Migrate applies every migration newer than the current schema version.
// Each migration runs in its own transaction. It returns the versions applied.
func Migrate(ctx context.Context, db *sql.DB, migrations []Migration) ([]int, error) {
	if _, err := db.ExecContext(ctx, createMigrationsTable); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return nil, fmt.Errorf("read schema version: %w", err)
	}

	var applied []int
	for _, m := range migrations {
		if m.Version <= current {
			continue
		}
		if err := apply(ctx, db, m); err != nil {
			return applied, fmt.Errorf("migration %d_%s: %w", m.Version, m.Name, err)
		}
		applied = append(applied, m.Version)
	}
	return applied, nil
}

func apply(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.Version, m.Name); err != nil {
		return err
	}
	return tx.Commit()
}
