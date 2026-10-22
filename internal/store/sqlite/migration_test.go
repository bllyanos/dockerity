package sqlite

import (
	"testing"
	"testing/fstest"
)

func TestLoadMigrationsSortsAndSkipsNonSQL(t *testing.T) {
	fsys := fstest.MapFS{
		"0002_b.sql": {Data: []byte("B")},
		"0001_a.sql": {Data: []byte("A")},
		"README.md":  {Data: []byte("x")},
		"10_c.sql":   {Data: []byte("C")},
	}
	got, err := LoadMigrations(fsys)
	if err != nil {
		t.Fatal(err)
	}
	want := []Migration{{1, "a", "A"}, {2, "b", "B"}, {10, "c", "C"}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLoadMigrationsRejectsBadNames(t *testing.T) {
	for _, name := range []string{"init.sql", "0001_.sql", "abc_x.sql", "0000_zero.sql", "-1_neg.sql"} {
		if _, err := LoadMigrations(fstest.MapFS{name: {Data: []byte("x")}}); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestLoadMigrationsRejectsDuplicateVersions(t *testing.T) {
	fsys := fstest.MapFS{"1_a.sql": {}, "0001_b.sql": {}}
	if _, err := LoadMigrations(fsys); err == nil {
		t.Fatal("expected duplicate version error")
	}
}

func TestEmbeddedMigrationsLoad(t *testing.T) {
	ms, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 || ms[0].Version != 1 {
		t.Fatalf("unexpected embedded migrations: %+v", ms)
	}
}
