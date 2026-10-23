package crypto

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseKey(t *testing.T) {
	k := mustKey(t)
	got, err := ParseKey("  " + k.Encode() + "\n")
	if err != nil || got != k {
		t.Fatalf("round trip: %v", err)
	}
	bad := map[string]string{
		"empty":      "",
		"not base64": "!!!not-base64!!!",
		"too short":  base64.StdEncoding.EncodeToString(make([]byte, 16)),
		"too long":   base64.StdEncoding.EncodeToString(make([]byte, 33)),
	}
	for name, s := range bad {
		if _, err := ParseKey(s); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestReadKeyFile(t *testing.T) {
	dir := t.TempDir()
	k := mustKey(t)
	good := filepath.Join(dir, "good")
	_ = os.WriteFile(good, []byte(k.Encode()+"\n"), 0o600)
	if got, err := ReadKeyFile(good); err != nil || got != k {
		t.Fatalf("good file: %v", err)
	}

	if _, err := ReadKeyFile(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing file accepted")
	}
	secretish := "c2VjcmV0LWJ1dC13cm9uZy1sZW5ndGg="
	bad := filepath.Join(dir, "bad")
	_ = os.WriteFile(bad, []byte(secretish), 0o600)
	_, err := ReadKeyFile(bad)
	if err == nil || strings.Contains(err.Error(), secretish) {
		t.Fatalf("bad file: %v", err)
	}
}

func TestKeyNeverPrinted(t *testing.T) {
	k := mustKey(t)
	for _, s := range []string{fmt.Sprint(k), fmt.Sprintf("%v %+v %#v %s", k, k, k, k)} {
		if strings.Contains(s, k.Encode()) || !strings.Contains(s, k.ID()) {
			t.Fatalf("formatted key leaks or lacks id: %q", s)
		}
	}
	if len(k.ID()) != 16 || k.ID() == mustKey(t).ID() {
		t.Fatal("unexpected key id")
	}
}
