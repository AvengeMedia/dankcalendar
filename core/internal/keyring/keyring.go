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

	kr "github.com/99designs/keyring"
	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
	"github.com/AvengeMedia/dankgo/log"
	"github.com/AvengeMedia/dankgo/portal"
)

const (
	serviceName            = "dankcal"
	keychainName           = "dankcal"
	fallbackCollectionName = "login"

	secretServiceBus  = "org.freedesktop.secrets"
	secretServicePath = "/org/freedesktop/secrets"
)

var ErrNotFound = errors.New("keyring: key not found")

type Store struct {
	ring          kr.Keyring
	secretService *secretServiceKeyring
	available     bool
}

func Open() *Store {
	cfg := defaultConfig()
	if portal.InFlatpak() {
		if _, err := portalSecret(); err != nil {
			log.Warnf("secret portal unavailable, falling back to encrypted db (%v)", err)
			return &Store{available: false}
		}
		cfg = flatpakConfig()
	} else {
		service, err := openSecretService(resolveDefaultCollection())
		if err == nil {
			if _, probeErr := service.Get("__dankcal_probe__"); probeErr == nil || errors.Is(probeErr, kr.ErrKeyNotFound) {
				log.Debugf("Secret Service keyring backend ready")
				return &Store{secretService: service, available: true}
			} else {
				err = probeErr
			}
		}
		log.Debugf("Secret Service keyring unavailable, trying fallback backends (%v)", err)
	}

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
		LibSecretCollectionName: resolveDefaultCollection(),
		KWalletAppID:            serviceName,
		KWalletFolder:           serviceName,
		FileDir:                 fileDir,
		FilePasswordFunc:        filePassword,
		AllowedBackends: []kr.BackendType{
			kr.KWalletBackend,
			kr.PassBackend,
			kr.FileBackend,
		},
	}
}

// resolveDefaultCollection reuses the Secret Service "default" collection
// (e.g. KWallet's existing wallet) instead of forcing a separate "login" one.
func resolveDefaultCollection() string {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fallbackCollectionName
	}

	var path dbus.ObjectPath
	obj := conn.Object(secretServiceBus, dbus.ObjectPath(secretServicePath))
	if err := obj.Call("org.freedesktop.Secret.Service.ReadAlias", 0, "default").Store(&path); err != nil {
		return fallbackCollectionName
	}

	name := collectionBaseName(string(path))
	if name == "" {
		return fallbackCollectionName
	}

	log.Debugf("keyring using default secret collection %q", name)
	return name
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
// object paths, matching how 99designs/keyring matches collections by name.
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

// flatpakConfig avoids org.freedesktop.secrets entirely: the sandbox has no
// talk permission for it, so the encrypted file backend is keyed with the
// per-app master secret from the XDG Secret portal instead.
func flatpakConfig() kr.Config {
	fileDir := ""
	if dir, err := paths.DataDir(); err == nil {
		fileDir = filepath.Join(dir, "keyring")
	}

	return kr.Config{
		ServiceName:      serviceName,
		FileDir:          fileDir,
		FilePasswordFunc: portalFilePassword,
		AllowedBackends:  []kr.BackendType{kr.FileBackend},
	}
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

func (s *Store) Available() bool { return s.available }

func (s *Store) Get(accountID, key string) ([]byte, error) {
	if !s.available {
		return nil, ErrNotFound
	}

	entry := entryKey(accountID, key)
	if s.secretService != nil {
		value, err := s.secretService.Get(entry)
		switch {
		case errors.Is(err, kr.ErrKeyNotFound):
			return nil, ErrNotFound
		case err != nil:
			return nil, fmt.Errorf("keyring get: %w", err)
		}
		return value, nil
	}

	item, err := s.ring.Get(entry)
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

	if s.secretService != nil {
		if err := s.secretService.Set(entryKey(accountID, key), value, "dankcal: "+accountID+" ("+key+")"); err != nil {
			return fmt.Errorf("keyring set: %w", err)
		}
		return nil
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

	if s.secretService != nil {
		err := s.secretService.Remove(entryKey(accountID, key))
		switch {
		case errors.Is(err, kr.ErrKeyNotFound):
			return nil
		case err != nil:
			return fmt.Errorf("keyring delete: %w", err)
		}
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
