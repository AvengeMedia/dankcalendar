package keyring

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
	"github.com/AvengeMedia/dankgo/log"
	"github.com/AvengeMedia/dankgo/portal"
)

const (
	serviceName                = "dankcal"
	fallbackCollectionName     = "login"
	credentialDescription      = "Dank Calendar credential"
	secretServiceBus           = "org.freedesktop.secrets"
	secretServicePath          = "/org/freedesktop/secrets"
	secretServiceInterface     = "org.freedesktop.Secret.Service"
	secretServiceCollectionIfc = "org.freedesktop.Secret.Collection"
	secretServiceItemIfc       = "org.freedesktop.Secret.Item"
	secretServicePromptIfc     = "org.freedesktop.Secret.Prompt"
	secretServicePromptTimeout = 30 * time.Second
)

var ErrNotFound = errors.New("keyring: key not found")

type Store struct {
	secret    *secretServiceStore
	file      *fileStore
	available bool
}

func Open() *Store {
	if portal.InFlatpak() {
		if _, err := portalSecret(); err != nil {
			log.Warnf("secret portal unavailable, falling back to encrypted db (%v)", err)
			return &Store{available: false}
		}

		store, err := openFileStore(portalFilePassword)
		if err != nil {
			log.Warnf("keyring unavailable, falling back to encrypted db (%v)", err)
			return &Store{available: false}
		}

		log.Debugf("keyring backend ready")
		return &Store{file: store, available: true}
	}

	secret, secretErr := openSecretServiceStore(resolveDefaultCollectionPath())
	if secretErr == nil {
		log.Debugf("keyring backend ready")
		return &Store{secret: secret, available: true}
	}

	store, fileErr := openFileStore(filePassword)
	if fileErr == nil {
		log.Warnf("system keyring unavailable, using encrypted local keyring (%v)", secretErr)
		log.Debugf("keyring backend ready")
		return &Store{file: store, available: true}
	}

	log.Warnf("keyring unavailable, falling back to encrypted db (%v)", errors.Join(secretErr, fileErr))
	return &Store{available: false}
}

func resolveDefaultCollectionPath() dbus.ObjectPath {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fallbackCollectionPath()
	}

	var path dbus.ObjectPath
	obj := conn.Object(secretServiceBus, dbus.ObjectPath(secretServicePath))
	if err := obj.Call(secretServiceInterface+".ReadAlias", 0, "default").Store(&path); err != nil {
		return fallbackCollectionPath()
	}
	if path == "" || path == "/" {
		return fallbackCollectionPath()
	}

	if name := collectionBaseName(string(path)); name != "" {
		log.Debugf("keyring using default secret collection %q", name)
	}
	return path
}

func fallbackCollectionPath() dbus.ObjectPath {
	return dbus.ObjectPath(secretServicePath + "/collection/" + fallbackCollectionName)
}

func collectionBaseName(path string) string {
	decoded := decodeCollectionPath(path)
	idx := strings.LastIndex(decoded, "/")
	if idx < 0 || idx == len(decoded)-1 {
		return ""
	}
	return decoded[idx+1:]
}

// decodeCollectionPath expands the "_XX" hex escapes the Secret Service uses in
// object paths.
func decodeCollectionPath(src string) string {
	var b strings.Builder
	for i := 0; i < len(src); i++ {
		if src[i] != '_' {
			b.WriteByte(src[i])
			continue
		}
		if i+3 > len(src) {
			return src
		}
		decoded, err := hex.DecodeString(src[i+1 : i+3])
		if err != nil {
			return src
		}
		b.Write(decoded)
		i += 2
	}
	return b.String()
}

func filePassword(prompt string) (string, error) {
	return "dankcal-local", nil
}

var portalSecret = sync.OnceValues(func() (string, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return "", fmt.Errorf("connect session bus: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := portal.RetrieveSecret(ctx, conn, portal.SecretOptions{})
	if err != nil {
		return "", err
	}
	if len(res.Secret) == 0 {
		return "", errors.New("secret portal returned an empty secret")
	}
	return hex.EncodeToString(res.Secret), nil
})

func portalFilePassword(prompt string) (string, error) {
	return portalSecret()
}

func entryKey(accountID, key string) string {
	return accountID + "::" + key
}

func entryLabel(accountID, key string) string {
	return "dankcal: " + accountID + " (" + key + ")"
}

func (s *Store) Available() bool { return s.available }

func (s *Store) Get(accountID, key string) ([]byte, error) {
	if !s.available {
		return nil, ErrNotFound
	}

	entry := entryKey(accountID, key)
	value, err := s.backendGet(entry)
	if err != nil {
		return nil, fmt.Errorf("keyring get: %w", err)
	}
	return value, nil
}

func (s *Store) Set(accountID, key string, value []byte) error {
	if !s.available {
		return ErrNotFound
	}

	if err := s.backendSet(entryKey(accountID, key), value, entryLabel(accountID, key)); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

func (s *Store) Delete(accountID, key string) error {
	if !s.available {
		return nil
	}

	err := s.backendDelete(entryKey(accountID, key))
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("keyring delete: %w", err)
	}
	return nil
}

func (s *Store) backendGet(key string) ([]byte, error) {
	switch {
	case s.secret != nil:
		return s.secret.Get(key)
	case s.file != nil:
		return s.file.Get(key)
	default:
		return nil, ErrNotFound
	}
}

func (s *Store) backendSet(key string, value []byte, label string) error {
	switch {
	case s.secret != nil:
		return s.secret.Set(key, value, label)
	case s.file != nil:
		return s.file.Set(key, value, label)
	default:
		return ErrNotFound
	}
}

func (s *Store) backendDelete(key string) error {
	switch {
	case s.secret != nil:
		return s.secret.Delete(key)
	case s.file != nil:
		return s.file.Delete(key)
	default:
		return ErrNotFound
	}
}

func keyringDir() (string, error) {
	dir, err := paths.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "keyring"), nil
}
