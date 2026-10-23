package crypto

import (
	"bytes"
	"errors"
	"testing"
)

func mustKey(t *testing.T) Key {
	t.Helper()
	k, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestSealOpen(t *testing.T) {
	k := mustKey(t)
	aad := []byte("env:1/v3")
	s, err := Seal(k, []byte("hunter2"), aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(s.Ciphertext, []byte("hunter2")) {
		t.Fatal("plaintext visible in ciphertext")
	}
	got, err := Open(k, s, aad)
	if err != nil || string(got) != "hunter2" {
		t.Fatalf("open: %q %v", got, err)
	}

	s2, _ := Seal(k, []byte("hunter2"), aad)
	if bytes.Equal(s.Nonce, s2.Nonce) || bytes.Equal(s.Ciphertext, s2.Ciphertext) {
		t.Fatal("nonces must be random")
	}
}

func TestOpenRejects(t *testing.T) {
	k := mustKey(t)
	aad := []byte("env:1/v3")
	s, _ := Seal(k, []byte("hunter2"), aad)

	flip := func(b []byte) []byte {
		c := bytes.Clone(b)
		c[0] ^= 1
		return c
	}
	cases := map[string]struct {
		key Key
		s   Sealed
		aad []byte
	}{
		"wrong key":           {mustKey(t), s, aad},
		"wrong aad":           {k, s, []byte("env:2/v3")},
		"tampered ciphertext": {k, Sealed{s.Nonce, flip(s.Ciphertext)}, aad},
		"tampered nonce":      {k, Sealed{flip(s.Nonce), s.Ciphertext}, aad},
		"short nonce":         {k, Sealed{s.Nonce[:4], s.Ciphertext}, aad},
		"truncated":           {k, Sealed{s.Nonce, s.Ciphertext[:3]}, aad},
	}
	for name, c := range cases {
		if _, err := Open(c.key, c.s, c.aad); !errors.Is(err, ErrDecrypt) {
			t.Errorf("%s: got %v, want ErrDecrypt", name, err)
		}
	}
}
