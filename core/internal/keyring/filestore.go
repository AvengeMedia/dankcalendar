package keyring

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	jose "github.com/dvsekhvalnov/jose2go"
	"github.com/mtibben/percent"

	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
)

// fileStore keeps the on-disk format of the 99designs/keyring file backend —
// one percent-escaped file per key holding a PBES2-HS256+A128KW/A256GCM JWE
// around a JSON item — so stores written before the rewrite keep decrypting.
type fileStore struct {
	dir      string
	password string
}

// storedSecret mirrors the JSON layout 99designs/keyring wrapped around
// values, both on disk and in Secret Service items.
type storedSecret struct {
	Key         string
	Data        []byte
	Label       string
	Description string
}

func openFileStore(password func() (string, error)) (*fileStore, error) {
	dir, err := paths.DataDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "keyring")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	pwd, err := password()
	if err != nil {
		return nil, err
	}
	return &fileStore{dir: dir, password: pwd}, nil
}

func (f *fileStore) Get(key string) ([]byte, error) {
	data, err := os.ReadFile(f.filename(key))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	}

	payload, _, err := jose.Decode(string(data), f.password)
	if err != nil {
		return nil, err
	}
	return decodeStoredSecret(key, []byte(payload)), nil
}

func (f *fileStore) Set(key string, value []byte, label string) error {
	payload, err := json.Marshal(storedSecret{Key: key, Data: value, Label: label, Description: credentialDescription})
	if err != nil {
		return err
	}

	token, err := jose.Encrypt(string(payload), jose.PBES2_HS256_A128KW, jose.A256GCM, f.password,
		jose.Headers(map[string]any{"created": time.Now().String()}))
	if err != nil {
		return err
	}
	return os.WriteFile(f.filename(key), []byte(token), 0o600)
}

func (f *fileStore) Delete(key string) error {
	err := os.Remove(f.filename(key))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	return err
}

func (f *fileStore) filename(key string) string {
	return filepath.Join(f.dir, percent.Encode(key, "/"))
}

func decodeStoredSecret(key string, payload []byte) []byte {
	var stored storedSecret
	if err := json.Unmarshal(payload, &stored); err == nil && stored.Key == key {
		return stored.Data
	}
	return payload
}
