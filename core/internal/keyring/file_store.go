package keyring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	jose "github.com/dvsekhvalnov/jose2go"
	"github.com/mtibben/percent"
)

type passwordFunc func(prompt string) (string, error)

type fileStore struct {
	dir          string
	passwordFunc passwordFunc
	password     string
}

type legacyStoredSecret struct {
	Key         string
	Data        []byte
	Label       string
	Description string
}

func openFileStore(passwordFn passwordFunc) (*fileStore, error) {
	dir, err := keyringDir()
	if err != nil {
		return nil, err
	}

	store := &fileStore{dir: dir, passwordFunc: passwordFn}
	if err := store.unlock(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *fileStore) Get(key string) ([]byte, error) {
	filename, err := s.filename(key)
	if err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(filename)
	switch {
	case os.IsNotExist(err):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	}

	if err := s.unlock(); err != nil {
		return nil, err
	}

	payload, _, err := jose.Decode(string(bytes), s.password)
	if err != nil {
		return nil, err
	}
	return decodeStoredSecret(key, []byte(payload)), nil
}

func (s *fileStore) Set(key string, value []byte, label string) error {
	payload, err := encodeStoredSecret(key, value, label)
	if err != nil {
		return err
	}

	if err := s.unlock(); err != nil {
		return err
	}

	token, err := jose.Encrypt(string(payload), jose.PBES2_HS256_A128KW, jose.A256GCM, s.password,
		jose.Headers(map[string]any{
			"created": time.Now().String(),
		}))
	if err != nil {
		return err
	}

	filename, err := s.filename(key)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, []byte(token), 0o600)
}

func (s *fileStore) Delete(key string) error {
	filename, err := s.filename(key)
	if err != nil {
		return err
	}
	if err := os.Remove(filename); os.IsNotExist(err) {
		return ErrNotFound
	} else {
		return err
	}
}

func (s *fileStore) resolveDir() (string, error) {
	if s.dir == "" {
		return "", fmt.Errorf("no directory provided for file keyring")
	}

	stat, err := os.Stat(s.dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(s.dir, 0o700)
	} else if err != nil && stat != nil && !stat.IsDir() {
		err = fmt.Errorf("%s is a file, not a directory", s.dir)
	}

	return s.dir, err
}

func (s *fileStore) unlock() error {
	if _, err := s.resolveDir(); err != nil {
		return err
	}

	if s.password != "" {
		return nil
	}

	pwd, err := s.passwordFunc(fmt.Sprintf("Enter passphrase to unlock %q", s.dir))
	if err != nil {
		return err
	}
	s.password = pwd
	return nil
}

func (s *fileStore) filename(key string) (string, error) {
	dir, err := s.resolveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, percent.Encode(key, "/")), nil
}

func encodeStoredSecret(key string, value []byte, label string) ([]byte, error) {
	return json.Marshal(legacyStoredSecret{
		Key:         key,
		Data:        value,
		Label:       label,
		Description: credentialDescription,
	})
}

func decodeStoredSecret(key string, payload []byte) []byte {
	var legacy legacyStoredSecret
	if err := json.Unmarshal(payload, &legacy); err == nil && legacy.Key == key {
		return legacy.Data
	}
	return payload
}
