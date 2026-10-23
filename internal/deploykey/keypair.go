// Package deploykey generates and stores per-app SSH deploy keys
// (docs/design.md section 6.1.1). Private keys live only in the secret store.
package deploykey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// KeyPair is a freshly generated deploy key.
type KeyPair struct {
	// Public is an authorized_keys line to add to the repo as a read-only deploy key.
	Public string
	// PrivatePEM is the OpenSSH private key. Never log or return it to clients.
	PrivatePEM []byte
}

// String hides the private key.
func (k KeyPair) String() string { return "deploykey.KeyPair{" + k.Public + "}" }

// GoString hides the private key from %#v.
func (k KeyPair) GoString() string { return k.String() }

// Generate creates an ed25519 key pair labeled with comment (e.g. "dockerity-shop").
func Generate(comment string) (KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return KeyPair{}, err
	}
	block, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return KeyPair{}, err
	}
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))) + " " + comment
	return KeyPair{Public: line, PrivatePEM: pem.EncodeToMemory(block)}, nil
}

// Signer parses a stored private key for SSH authentication.
func Signer(privatePEM []byte) (ssh.Signer, error) {
	s, err := ssh.ParsePrivateKey(privatePEM)
	if err != nil {
		return nil, fmt.Errorf("parse deploy key: %w", err)
	}
	return s, nil
}
