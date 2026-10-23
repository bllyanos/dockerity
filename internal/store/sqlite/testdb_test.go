package sqlite

import (
	"context"
	"database/sql"
	"testing"
)

// migratedDB returns a fresh database with all migrations applied.
func migratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTestDB(t)
	ms, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), db, ms); err != nil {
		t.Fatal(err)
	}
	return db
}
