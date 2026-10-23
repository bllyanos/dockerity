package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// ErrDecrypt means a ciphertext could not be authenticated: wrong key,
// wrong additional data, or tampered ciphertext or nonce.
var ErrDecrypt = errors.New("decryption failed")

// Sealed is an AES-256-GCM ciphertext with its nonce.
type Sealed struct {
	Nonce      []byte
	Ciphertext []byte
}

// Seal encrypts plaintext with key and a fresh random nonce. aad is
// authenticated but not encrypted; use it to bind the ciphertext to its owner.
func Seal(key Key, plaintext, aad []byte) (Sealed, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return Sealed{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return Sealed{}, err
	}
	return Sealed{Nonce: nonce, Ciphertext: gcm.Seal(nil, nonce, plaintext, aad)}, nil
}

// Open decrypts s. It returns ErrDecrypt when authentication fails.
func Open(key Key, s Sealed, aad []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(s.Nonce) != gcm.NonceSize() {
		return nil, ErrDecrypt
	}
	plaintext, err := gcm.Open(nil, s.Nonce, s.Ciphertext, aad)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

func newGCM(key Key) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
