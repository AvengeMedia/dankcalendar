package keyring

import (
	"errors"
	"fmt"
)

type SecretRef struct {
	AccountID string
	Key       string
}

// MigrateLoginCollection moves the given secrets out of the legacy "login"
// Secret Service collection into the resolved default one. It is a no-op when
// login is already the default, no refs are given, or an entry isn't present.
func (s *Store) MigrateLoginCollection(refs []SecretRef) (int, error) {
	service, ok := s.backend.(*secretService)
	if !ok || len(refs) == 0 {
		return 0, nil
	}
	if service.collection == loginCollectionPath {
		return 0, nil
	}

	migrated := 0
	for _, ref := range refs {
		key := entryKey(ref.AccountID, ref.Key)
		value, err := service.getFrom(loginCollectionPath, key)
		switch {
		case errors.Is(err, ErrNotFound):
			continue
		case err != nil:
			return migrated, fmt.Errorf("read %s from login: %w", key, err)
		}

		if err := s.Set(ref.AccountID, ref.Key, value); err != nil {
			return migrated, err
		}
		_ = service.deleteFrom(loginCollectionPath, key)
		migrated++
	}
	return migrated, nil
}
