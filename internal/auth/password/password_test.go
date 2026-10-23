package password

import (
	"errors"
	"strings"
	"testing"
)

// fast keeps tests quick; production uses DefaultParams.
var fast = Params{Memory: 1024, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := Hash("correct horse battery", fast)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=1024,t=1,p=1$") {
		t.Fatalf("unexpected encoding %q", h)
	}
	if err := Verify("correct horse battery", h); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := Verify("wrong horse battery", h); !errors.Is(err, ErrMismatch) {
		t.Fatalf("got %v, want ErrMismatch", err)
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := Hash("same password!", fast)
	b, _ := Hash("same password!", fast)
	if a == b {
		t.Fatal("two hashes of the same password must differ")
	}
}

func TestVerifyHonorsStoredParams(t *testing.T) {
	stronger := fast
	stronger.Time = 2
	h, _ := Hash("parameters matter", stronger)
	if !strings.Contains(h, "t=2") {
		t.Fatalf("params not encoded: %q", h)
	}
	if err := Verify("parameters matter", h); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyMalformed(t *testing.T) {
	good, _ := Hash("some password", fast)
	parts := strings.Split(good, "$")
	cases := []string{
		"",
		"plain",
		"$argon2i$v=19$m=1024,t=1,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=18$m=1024,t=1,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=0,t=1,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=1024,t=1,p=1$!!!$" + parts[5],
		"$argon2id$v=19$m=1024,t=1,p=1$" + parts[4] + "$",
		good + "$extra",
	}
	for _, c := range cases {
		if err := Verify("some password", c); !errors.Is(err, ErrMalformedHash) {
			t.Errorf("%q: got %v, want ErrMalformedHash", c, err)
		}
	}
}

func TestVerifyTamperedKey(t *testing.T) {
	h, _ := Hash("some password", fast)
	parts := strings.Split(h, "$")
	other, _ := Hash("other password", fast)
	parts[5] = strings.Split(other, "$")[5]
	if err := Verify("some password", strings.Join(parts, "$")); !errors.Is(err, ErrMismatch) {
		t.Fatalf("got %v, want ErrMismatch", err)
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("short"); err == nil {
		t.Error("short password accepted")
	}
	if err := Validate(strings.Repeat("é", MinLength)); err != nil {
		t.Errorf("multi-byte password of MinLength rejected: %v", err)
	}
}
