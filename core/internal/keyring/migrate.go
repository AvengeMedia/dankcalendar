package keyring

import (
	"errors"
	"fmt"

	kr "github.com/99designs/keyring"
)

const legacyLoginCollection = "login"

type SecretRef struct {
	AccountID string
	Key       string
}

// MigrateLoginCollection moves the given secrets out of the legacy "login"
// Secret Service collection into the resolved default one. It is a no-op when
// login is already the default, no refs are given, or an entry isn't present.
func (s *Store) MigrateLoginCollection(refs []SecretRef) (int, error) {
	if !s.available || len(refs) == 0 {
		return 0, nil
	}
	if resolveDefaultCollection() == legacyLoginCollection {
		return 0, nil
	}

	login, err := kr.Open(kr.Config{
		ServiceName:             serviceName,
		LibSecretCollectionName: legacyLoginCollection,
		AllowedBackends:         []kr.BackendType{kr.SecretServiceBackend},
	})
	if err != nil {
		return 0, fmt.Errorf("open legacy login collection: %w", err)
	}

	migrated := 0
	for _, ref := range refs {
		ek := entryKey(ref.AccountID, ref.Key)
		item, err := login.Get(ek)
		switch {
		case errors.Is(err, kr.ErrKeyNotFound):
			continue
		case err != nil:
			return migrated, fmt.Errorf("read %s from login: %w", ek, err)
		}

		if err := s.Set(ref.AccountID, ref.Key, item.Data); err != nil {
			return migrated, err
		}
		_ = login.Remove(ek)
		migrated++
	}
	return migrated, nil
}
