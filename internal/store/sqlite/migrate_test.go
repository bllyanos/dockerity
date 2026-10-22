package sqlite

import (
	"context"
	"database/sql"
	"slices"
	"testing"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenEnablesPragmas(t *testing.T) {
	db := openTestDB(t)
	var mode string
	var fk int
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" || fk != 1 {
		t.Fatalf("journal_mode=%s foreign_keys=%d", mode, fk)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	ms, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}

	first, err := Migrate(ctx, db, ms)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(ms) {
		t.Fatalf("applied %v, want all %d", first, len(ms))
	}
	second, err := Migrate(ctx, db, ms)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("second run applied %v", second)
	}
}

func TestMigrateAppliesOnlyNewer(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	m1 := Migration{1, "a", `CREATE TABLE a (id INTEGER)`}
	m2 := Migration{2, "b", `CREATE TABLE b (id INTEGER)`}

	if _, err := Migrate(ctx, db, []Migration{m1}); err != nil {
		t.Fatal(err)
	}
	got, err := Migrate(ctx, db, []Migration{m1, m2})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []int{2}) {
		t.Fatalf("applied %v, want [2]", got)
	}
}

func TestMigrateRollsBackFailedMigration(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	ms := []Migration{
		{1, "ok", `CREATE TABLE a (id INTEGER)`},
		{2, "bad", `CREATE TABLE b (id INTEGER); THIS IS NOT SQL`},
	}

	applied, err := Migrate(ctx, db, ms)
	if err == nil {
		t.Fatal("expected error")
	}
	if !slices.Equal(applied, []int{1}) {
		t.Fatalf("applied %v, want [1]", applied)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'b'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("table b should have been rolled back")
	}
	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("schema version %d, want 1", version)
	}
}
