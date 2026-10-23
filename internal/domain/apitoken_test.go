package domain

import (
	"strings"
	"testing"
)

func TestTokenName(t *testing.T) {
	if got := NormalizeTokenName("  ci deploy  "); got != "ci deploy" {
		t.Fatalf("normalize: %q", got)
	}
	for _, ok := range []string{"x", strings.Repeat("é", 64)} {
		if err := ValidateTokenName(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", strings.Repeat("a", 65)} {
		if err := ValidateTokenName(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
