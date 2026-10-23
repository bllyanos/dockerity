package secrets

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestValidateKey(t *testing.T) {
	for _, ok := range []string{"A", "_X", "DATABASE_URL", "K8S_TOKEN_2"} {
		if err := ValidateKey(ok); err != nil {
			t.Errorf("%s rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "1A", "lower", "WITH-DASH", "WITH SPACE", strings.Repeat("A", MaxKeyLength+1)} {
		if err := ValidateKey(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestApply(t *testing.T) {
	current := map[string]string{"A": "1", "B": "2"}
	next, err := Changes{Set: map[string]string{"B": "20", "C": "3"}, Unset: []string{"A"}}.Apply(current)
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(next, map[string]string{"B": "20", "C": "3"}) {
		t.Fatalf("got %v", next)
	}
	if current["A"] != "1" {
		t.Fatal("current must not be modified")
	}
}

func TestApplyRejects(t *testing.T) {
	current := map[string]string{"A": "hunter2"}
	cases := map[string]Changes{
		"empty":         {},
		"bad key":       {Set: map[string]string{"bad": "x"}},
		"set and unset": {Set: map[string]string{"A": "x"}, Unset: []string{"A"}},
		"unset missing": {Unset: []string{"MISSING"}},
		"value too big": {Set: map[string]string{"BIG": strings.Repeat("x", MaxValueLength+1)}},
	}
	for name, c := range cases {
		_, err := c.Apply(current)
		var ve *domain.ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: got %v", name, err)
			continue
		}
		if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: error leaks a value", name)
		}
	}
}

func TestAuditMetadata(t *testing.T) {
	got := Changes{Set: map[string]string{"B": "secret-b", "A": "secret-a"}, Unset: []string{"C"}}.AuditMetadata()
	if got["set_keys"] != "[A B]" || got["unset_keys"] != "[C]" {
		t.Fatalf("got %v", got)
	}
	for _, v := range got {
		if strings.Contains(v, "secret-") {
			t.Fatal("metadata leaks a value")
		}
	}
}
