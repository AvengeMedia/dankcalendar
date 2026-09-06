package keyring

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type lockedBackend struct{}

func (lockedBackend) Get(string) ([]byte, error)       { return nil, ErrLocked }
func (lockedBackend) Set(string, []byte, string) error { return nil }
func (lockedBackend) Delete(string) error              { return ErrNotFound }

type mapFallback map[string][]byte

func (m mapFallback) Get(_ context.Context, accountID, key string) ([]byte, error) {
	value, ok := m[entryKey(accountID, key)]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}

func (m mapFallback) Set(_ context.Context, accountID, key string, value []byte) error {
	m[entryKey(accountID, key)] = value
	return nil
}

func (m mapFallback) Delete(_ context.Context, accountID, key string) error {
	delete(m, entryKey(accountID, key))
	return nil
}

func TestLockedKeyringIsNotMissing(t *testing.T) {
	store := &Store{backend: lockedBackend{}}

	_, err := store.Get("acc", "token")
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("Store.Get() error = %v, want ErrLocked", err)
	}

	secrets := NewSecretStore(store, mapFallback{})
	_, err = secrets.Get(context.Background(), "acc", "token")
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("SecretStore.Get() error = %v, want ErrLocked", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("a locked keyring must not read as a missing credential")
	}
}

func TestLockedKeyringStillServesFallback(t *testing.T) {
	fallback := mapFallback{entryKey("acc", "token"): []byte("db-token")}
	secrets := NewSecretStore(&Store{backend: lockedBackend{}}, fallback)

	got, err := secrets.Get(context.Background(), "acc", "token")
	if err != nil {
		t.Fatalf("SecretStore.Get() error = %v", err)
	}
	if !bytes.Equal(got, []byte("db-token")) {
		t.Fatalf("SecretStore.Get() = %q, want fallback value", got)
	}
}
