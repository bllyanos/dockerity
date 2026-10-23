package buildlog

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestExpired(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	files := []File{
		{ID: 3, ModTime: now.Add(-40 * 24 * time.Hour)},
		{ID: 1, ModTime: now.Add(-31 * 24 * time.Hour)},
		{ID: 2, ModTime: now.Add(-29 * 24 * time.Hour)},
		{ID: 4, ModTime: now.Add(-30 * 24 * time.Hour)}, // exactly at the cutoff: kept
	}
	if got := Expired(files, now.Add(-30*24*time.Hour)); !slices.Equal(got, []int64{1, 3}) {
		t.Fatalf("got %v", got)
	}
}

func TestStoreExpire(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	old := time.Now().Add(-48 * time.Hour)
	for _, id := range []int64{1, 2} {
		w, err := s.Open(id)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("hi"))
		_ = w.Close()
	}
	_ = os.Chtimes(s.path(1), old, old)
	_ = os.WriteFile(filepath.Join(s.Dir, "notes.txt"), nil, 0o600)

	ids, err := s.Expire(time.Now().Add(-24 * time.Hour))
	if err != nil || !slices.Equal(ids, []int64{1}) {
		t.Fatalf("expired %v %v", ids, err)
	}
	if data, _ := s.Read(1); data != nil {
		t.Fatal("old log kept")
	}
	if data, _ := s.Read(2); string(data) != "hi" {
		t.Fatal("recent log removed")
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "notes.txt")); err != nil {
		t.Fatal("unrelated file removed")
	}
	if ids, err := (Store{Dir: filepath.Join(s.Dir, "missing")}).Expire(time.Now()); err != nil || len(ids) != 0 {
		t.Fatalf("missing dir: %v %v", ids, err)
	}
}
