package keyring

import "fmt"

type SecretRef struct {
	AccountID string
	Key       string
}

// MigrateLoginCollection moves the given secrets out of the legacy "login"
// Secret Service collection into the resolved default one. It is a no-op when
// login is already the default, no refs are given, or an entry isn't present.
func (s *Store) MigrateLoginCollection(refs []SecretRef) (int, error) {
	if !s.available || s.secret == nil || len(refs) == 0 {
		return 0, nil
	}
	migrated, err := s.secret.MigrateLoginCollection(refs)
	if err != nil {
		return migrated, fmt.Errorf("migrate login collection: %w", err)
	}
	return migrated, nil
}
