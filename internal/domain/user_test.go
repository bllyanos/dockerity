package domain

import (
	"errors"
	"testing"
)

func TestUsername(t *testing.T) {
	valid := []string{"bob", "alice.smith", "dev-ops_1", "a23456789012345678901234567890ab"}
	invalid := []string{"", "ab", "1bob", "_bob", "Bob", "bob smith", "bob@x", "a234567890123456789012345678901ab"}
	for _, s := range valid {
		if err := ValidateUsername(s); err != nil {
			t.Errorf("%q rejected: %v", s, err)
		}
	}
	for _, s := range invalid {
		err := ValidateUsername(s)
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Field != "username" {
			t.Errorf("%q: got %v, want username ValidationError", s, err)
		}
	}
	if got := NormalizeUsername("  Bob.Smith "); got != "bob.smith" {
		t.Errorf("NormalizeUsername = %q", got)
	}
}

func TestSentinelErrors(t *testing.T) {
	nf := NotFoundf("app %d not found", 3)
	if !errors.Is(nf, ErrNotFound) || nf.Error() != "app 3 not found" {
		t.Fatalf("NotFoundf: %v", nf)
	}
	c := Conflictf("name %q taken", "x")
	if !errors.Is(c, ErrConflict) || errors.Is(c, ErrNotFound) || c.Error() != `name "x" taken` {
		t.Fatalf("Conflictf: %v", c)
	}
}
