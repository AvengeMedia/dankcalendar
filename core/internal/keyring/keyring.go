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

var (
	ErrNotFound = errors.New("keyring: key not found")
	ErrLocked   = errors.New("keyring: collection is locked")
)

type backend interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, label string) error
	Delete(key string) error
}

type Store struct {
	mu           sync.Mutex
	backend      backend
	fileFallback backend
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
		file, fileErr := openFileStore(localFilePassword)
		if fileErr != nil {
			return &Store{backend: service}
		}
		return &Store{backend: service, fileFallback: file}
	}

	file, fileErr := openFileStore(localFilePassword)
	if fileErr != nil {
		log.Warnf("keyring unavailable, falling back to encrypted db (%v)", errors.Join(serviceErr, fileErr))
		return &Store{}
	}

	log.Warnf("secret service unavailable, using local encrypted keyring (%v)", serviceErr)
	return &Store{backend: file}
}

func (s *Store) Available() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend != nil
}

func (s *Store) Get(accountID, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil {
		return nil, ErrNotFound
	}

	value, err := s.backend.Get(entryKey(accountID, key))
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrLocked):
		if upgraded, uerr := s.maybeUpgradeLocked(); uerr == nil && upgraded {
			value, err = s.backend.Get(entryKey(accountID, key))
			if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrLocked) {
				if fb, ferr := s.fileGetLocked(entryKey(accountID, key)); ferr == nil {
					return fb, nil
				}
				return nil, fmt.Errorf("keyring get: %w", err)
			}
			if err == nil {
				return value, nil
			}
		}
		// Credentials written to the file store before an upgrade stay
		// readable; a locked service cannot prove absence.
		if fb, ferr := s.fileGetLocked(entryKey(accountID, key)); ferr == nil {
			return fb, nil
		}
		if errors.Is(err, ErrLocked) {
			return nil, err
		}
		return nil, ErrNotFound
	case err != nil:
		if upgraded, uerr := s.maybeUpgradeLocked(); uerr == nil && upgraded {
			if value, rerr := s.backend.Get(entryKey(accountID, key)); rerr == nil {
				return value, nil
			}
		}
		if fb, ferr := s.fileGetLocked(entryKey(accountID, key)); ferr == nil {
			return fb, nil
		}
		return nil, fmt.Errorf("keyring get: %w", err)
	}
	return value, nil
}

// fileGetLocked reads the retained pre-upgrade file store, if any.
// Callers must hold s.mu.
func (s *Store) fileGetLocked(key string) ([]byte, error) {
	if s.fileFallback == nil {
		return nil, ErrNotFound
	}
	return s.fileFallback.Get(key)
}

// maybeUpgradeLocked swaps a file fallback for the Secret Service when it
// appears after startup (e.g. the daemon launched before KeePassXC owned
// org.freedesktop.secrets). The previous file backend is retained as a read
// fallback so already-stored credentials are not stranded. True means the
// backend changed and the caller should retry. Callers must hold s.mu.
func (s *Store) maybeUpgradeLocked() (bool, error) {
	if portal.InFlatpak() {
		return false, nil
	}
	// Only file backends upgrade: test doubles and the Secret Service
	// itself must never trigger a bus dial here (the latter keeps unit
	// tests hermetic).
	if _, ok := s.backend.(*fileStore); !ok {
		return false, nil
	}
	service, err := openSecretService()
	if err != nil {
		return false, err
	}
	log.Debugf("keyring: secret service appeared, keeping local file fallback readable for %q",
		collectionBaseName(string(service.collection)))
	if s.backend != nil {
		s.fileFallback = s.backend
	}
	s.backend = service
	return true, nil
}

func (s *Store) Set(accountID, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil {
		return ErrNotFound
	}

	// Write to the Secret Service once it appears so token refreshes do not
	// split secrets between the file fallback and KeePassXC (issue #105).
	_, _ = s.maybeUpgradeLocked()
	if err := s.backend.Set(entryKey(accountID, key), value, entryLabel(accountID, key)); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	// Converge: the service copy is now canonical, drop the pre-upgrade one.
	if s.fileFallback != nil {
		_ = s.fileFallback.Delete(entryKey(accountID, key))
	}
	return nil
}

func (s *Store) Delete(accountID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil {
		return nil
	}

	if s.fileFallback != nil {
		_ = s.fileFallback.Delete(entryKey(accountID, key))
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
