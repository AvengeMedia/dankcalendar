package keyring

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const encPrefix = "dc1:"

// EncryptedSecretStore wraps a SecretStore and AES-GCM encrypts values with a
// key derived from the XDG Secret portal master secret (Flatpak).
type EncryptedSecretStore struct {
	inner calendar.SecretStore
	aead  cipher.AEAD
}

func NewEncryptedSecretStore(inner calendar.SecretStore, master []byte) (*EncryptedSecretStore, error) {
	key, err := deriveKey(master)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &EncryptedSecretStore{inner: inner, aead: aead}, nil
}

func deriveKey(master []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, master, []byte("dankcal"), []byte("secret-store-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	return key, nil
}

func (s *EncryptedSecretStore) Get(ctx context.Context, accountID, key string) ([]byte, error) {
	raw, err := s.inner.Get(ctx, accountID, key)
	if err != nil {
		return nil, err
	}
	return s.decrypt(raw)
}

func (s *EncryptedSecretStore) Set(ctx context.Context, accountID, key string, value []byte) error {
	enc, err := s.encrypt(value)
	if err != nil {
		return err
	}
	return s.inner.Set(ctx, accountID, key, enc)
}

func (s *EncryptedSecretStore) Delete(ctx context.Context, accountID, key string) error {
	return s.inner.Delete(ctx, accountID, key)
}

func (s *EncryptedSecretStore) encrypt(plain []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := s.aead.Seal(nonce, nonce, plain, nil)
	return append([]byte(encPrefix), out...), nil
}

func (s *EncryptedSecretStore) decrypt(raw []byte) ([]byte, error) {
	prefix := []byte(encPrefix)
	if len(raw) < len(prefix) || string(raw[:len(prefix)]) != encPrefix {
		// Plaintext leftovers from pre-portal installs: return as-is so the
		// next Set re-encrypts them.
		return raw, nil
	}
	payload := raw[len(prefix):]
	ns := s.aead.NonceSize()
	if len(payload) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := payload[:ns], payload[ns:]
	return s.aead.Open(nil, nonce, ct, nil)
}
