package token

import (
	"strings"
	"testing"
)

func TestNewIsUniqueAndPrefixed(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		tok, err := New(PrefixAPI)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(tok, PrefixAPI) || len(tok) != len(PrefixAPI)+43 {
			t.Fatalf("unexpected token %q", tok)
		}
		if seen[tok] {
			t.Fatal("duplicate token")
		}
		seen[tok] = true
	}
}

func TestDigest(t *testing.T) {
	a := Digest("dkta_abc")
	if a != Digest("dkta_abc") {
		t.Fatal("digest must be stable")
	}
	if a == Digest("dkta_abd") {
		t.Fatal("different tokens must have different digests")
	}
	if len(a) != 64 || strings.Contains(a, "abc") {
		t.Fatalf("unexpected digest %q", a)
	}
}

func TestHasPrefix(t *testing.T) {
	if !HasPrefix("dkts_x", PrefixSession) || HasPrefix("dkts_", PrefixSession) || HasPrefix("dkta_x", PrefixSession) {
		t.Fatal("HasPrefix mismatch")
	}
}
