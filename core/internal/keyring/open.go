package keyring

import (
	"fmt"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/repo"
	"github.com/AvengeMedia/dankgo/log"
)

// OpenSecretStore builds the credential store for the current environment.
// Flatpak uses the Secret portal master key + encrypted SQLite values so the
// sandbox does not need org.freedesktop.secrets. Native builds keep the
// keyring with an encrypted-db fallback.
func OpenSecretStore(r *repo.Repo) (calendar.SecretStore, error) {
	fallback := repo.NewSecretStore(r)
	if !InFlatpak() {
		return NewSecretStore(Open(), fallback), nil
	}

	master, err := RetrievePortalSecret()
	if err != nil {
		return nil, fmt.Errorf("flatpak secret portal: %w", err)
	}
	enc, err := NewEncryptedSecretStore(fallback, master)
	if err != nil {
		return nil, err
	}
	log.Debugf("using Secret portal for credential encryption")
	return enc, nil
}
