package buildlog

import (
	"io"
	"strings"
	"testing"
)

func TestStoreAppendAndRead(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if data, err := s.Read(1); err != nil || len(data) != 0 {
		t.Fatalf("missing log: %q %v", data, err)
	}
	w, err := s.Open(1)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(w, "line one\n")
	_ = w.Close()
	w, _ = s.Open(1)
	_, _ = io.WriteString(w, "line two\n")
	_ = w.Close()

	data, _ := s.Read(1)
	if string(data) != "line one\nline two\n" {
		t.Fatalf("got %q", data)
	}
	if err := s.Remove(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(1); err != nil {
		t.Fatalf("second remove: %v", err)
	}
}

func TestStoreCapsSize(t *testing.T) {
	s := Store{Dir: t.TempDir(), MaxBytes: 10}
	w, _ := s.Open(2)
	for range 5 {
		if n, err := io.WriteString(w, "123456"); err != nil || n != 6 {
			t.Fatalf("write: %d %v", n, err)
		}
	}
	_ = w.Close()
	data, _ := s.Read(2)
	if !strings.HasPrefix(string(data), "1234561234") || strings.Count(string(data), "log truncated") != 1 {
		t.Fatalf("got %q", data)
	}

	w, _ = s.Open(2)
	_, _ = io.WriteString(w, "more")
	_ = w.Close()
	if after, _ := s.Read(2); len(after) != len(data) {
		t.Fatal("a capped log must not grow when reopened")
	}
}
