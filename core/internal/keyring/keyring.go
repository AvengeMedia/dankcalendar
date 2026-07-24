package keyring

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/log"
	"github.com/AvengeMedia/dankgo/portal"
)

const (
	credentialDescription = "Dank Calendar credential"

	secretServiceBus    = "org.freedesktop.secrets"
	secretServicePath   = "/org/freedesktop/secrets"
	serviceInterface    = "org.freedesktop.Secret.Service"
	collectionInterface = "org.freedesktop.Secret.Collection"
	itemInterface       = "org.freedesktop.Secret.Item"
	promptInterface     = "org.freedesktop.Secret.Prompt"
	sessionInterface    = "org.freedesktop.Secret.Session"

	loginCollectionPath = dbus.ObjectPath("/org/freedesktop/secrets/collection/login")

	// promptTimeout bounds how long a Secret Service prompt may stay open; it
	// includes the user typing a keyring or KeePassXC master password.
	promptTimeout = 2 * time.Minute
)

var ErrNotFound = errors.New("keyring: key not found")

type backend interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, label string) error
	Delete(key string) error
}

type Store struct {
	backend backend
}

func Open() *Store {
	if portal.InFlatpak() {
		// The sandbox has no org.freedesktop.secrets talk permission; the
		// encrypted file store is keyed with the per-app master secret from
		// the XDG Secret portal instead.
		file, err := openFileStore(portalSecret)
		if err != nil {
			log.Warnf("secret portal unavailable, falling back to encrypted db (%v)", err)
			return &Store{}
		}
		return &Store{backend: file}
	}

	service, serviceErr := openSecretService()
	if serviceErr == nil {
		log.Debugf("keyring using secret collection %q", collectionBaseName(string(service.collection)))
		return &Store{backend: service}
	}

	file, fileErr := openFileStore(localFilePassword)
	if fileErr != nil {
		log.Warnf("keyring unavailable, falling back to encrypted db (%v)", errors.Join(serviceErr, fileErr))
		return &Store{}
	}

	log.Warnf("secret service unavailable, using local encrypted keyring (%v)", serviceErr)
	return &Store{backend: file}
}

func (s *Store) Available() bool { return s.backend != nil }

func (s *Store) Get(accountID, key string) ([]byte, error) {
	if s.backend == nil {
		return nil, ErrNotFound
	}

	value, err := s.backend.Get(entryKey(accountID, key))
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("keyring get: %w", err)
	}
	return value, nil
}

func (s *Store) Set(accountID, key string, value []byte) error {
	if s.backend == nil {
		return ErrNotFound
	}

	if err := s.backend.Set(entryKey(accountID, key), value, entryLabel(accountID, key)); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

func (s *Store) Delete(accountID, key string) error {
	if s.backend == nil {
		return nil
	}

	err := s.backend.Delete(entryKey(accountID, key))
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("keyring delete: %w", err)
	}
	return nil
}

func entryKey(accountID, key string) string {
	return accountID + "::" + key
}

func entryLabel(accountID, key string) string {
	return "dankcal: " + accountID + " (" + key + ")"
}

func collectionBaseName(path string) string {
	decoded := decodeCollectionPath(path)
	idx := strings.LastIndex(decoded, "/")
	if idx < 0 || idx == len(decoded)-1 {
		return ""
	}
	return decoded[idx+1:]
}

// decodeCollectionPath expands the "_XX" hex escapes the Secret Service uses
// in object paths.
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

func localFilePassword() (string, error) {
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
