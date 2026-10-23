package domain

import (
	"strings"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	for _, ok := range []string{"", "shop.example.com", "a.b", "staging.shop.localhost", "x-1.example.io"} {
		if err := ValidateDomain(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"localhost", "Shop.example.com", "-a.example.com", "a..b", "a_b.com", "https://a.com", "a.com/", strings.Repeat("a", 64) + ".com"} {
		if err := ValidateDomain(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestEphemeralDomain(t *testing.T) {
	got, err := EphemeralDomain("pr-1", "shop", "apps.example.com")
	if err != nil || got != "pr-1-shop.apps.example.com" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := EphemeralDomain(strings.Repeat("a", 32), strings.Repeat("b", 32), "x.y"); err == nil {
		t.Fatal("a 65-character label must be rejected")
	}
}
