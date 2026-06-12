package keyring

import (
	"errors"
	"fmt"
	"path/filepath"

	kr "github.com/99designs/keyring"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
)

const (
	serviceName    = "dankcal"
	keychainName   = "dankcal"
	collectionName = "login"
)

var ErrNotFound = errors.New("keyring: key not found")

type Store struct {
	ring      kr.Keyring
	available bool
}

func Open() *Store {
	cfg := defaultConfig()
	ring, err := kr.Open(cfg)
	if err != nil {
		log.Warnf("keyring unavailable, falling back to encrypted db (%v)", err)
		return &Store{available: false}
	}

	if _, probeErr := ring.Get("__dankcal_probe__"); probeErr != nil && !errors.Is(probeErr, kr.ErrKeyNotFound) {
		log.Warnf("keyring probe failed, falling back to encrypted db (%v)", probeErr)
		return &Store{available: false}
	}

	log.Debugf("keyring backend ready")
	return &Store{ring: ring, available: true}
}

func defaultConfig() kr.Config {
	fileDir := ""
	if dir, err := paths.DataDir(); err == nil {
		fileDir = filepath.Join(dir, "keyring")
	}

	return kr.Config{
		ServiceName:             serviceName,
		KeychainName:            keychainName,
		LibSecretCollectionName: collectionName,
		KWalletAppID:            serviceName,
		KWalletFolder:           serviceName,
		FileDir:                 fileDir,
		FilePasswordFunc:        filePassword,
		AllowedBackends: []kr.BackendType{
			kr.SecretServiceBackend,
			kr.KWalletBackend,
			kr.PassBackend,
			kr.FileBackend,
		},
	}
}

func filePassword(prompt string) (string, error) {
	return "dankcal-local", nil
}

func entryKey(accountID, key string) string {
	return accountID + "::" + key
}

func (s *Store) Available() bool { return s.available }

func (s *Store) Get(accountID, key string) ([]byte, error) {
	if !s.available {
		return nil, ErrNotFound
	}

	item, err := s.ring.Get(entryKey(accountID, key))
	switch {
	case errors.Is(err, kr.ErrKeyNotFound):
		return nil, ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("keyring get: %w", err)
	}
	return item.Data, nil
}

func (s *Store) Set(accountID, key string, value []byte) error {
	if !s.available {
		return ErrNotFound
	}

	err := s.ring.Set(kr.Item{
		Key:         entryKey(accountID, key),
		Data:        value,
		Label:       "dankcal: " + accountID + " (" + key + ")",
		Description: "Dank Calendar credential",
	})
	if err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

func (s *Store) Delete(accountID, key string) error {
	if !s.available {
		return nil
	}

	err := s.ring.Remove(entryKey(accountID, key))
	switch {
	case errors.Is(err, kr.ErrKeyNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("keyring delete: %w", err)
	}
	return nil
}
