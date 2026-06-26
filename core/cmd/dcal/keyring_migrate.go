package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	dankkeyring "github.com/AvengeMedia/dankcalendar/core/internal/keyring"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/caldav"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/google"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/ical"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/microsoft"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

// migrateLegacyLoginKeyring is a one-time fix for installs that stored
// credentials in a "login" collection before we resolved the Secret Service
// default. It runs once, only when existing accounts are present, then stamps a
// marker so new users and later boots skip it entirely.
func migrateLegacyLoginKeyring(ctx context.Context, store *dankkeyring.Store, r *repo.Repo) {
	if !store.Available() {
		return
	}

	marker, err := loginMigrationMarker()
	if err != nil {
		log.Warnf("keyring migration: %v", err)
		return
	}
	if _, err := os.Stat(marker); err == nil {
		return
	}

	accounts, err := r.ListAccounts(ctx)
	if err != nil {
		log.Warnf("keyring migration: list accounts: %v", err)
		return
	}

	moved, err := store.MigrateLoginCollection(loginSecretRefs(accounts))
	if err != nil {
		log.Warnf("keyring migration from login collection failed: %v", err)
		return
	}
	if moved > 0 {
		log.Infof("migrated %d credential(s) from the legacy login keyring", moved)
	}

	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		log.Warnf("keyring migration: write marker: %v", err)
	}
}

func loginMigrationMarker() (string, error) {
	dir, err := paths.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "keyring-login-migrated"), nil
}

func loginSecretRefs(accounts []*ent.Account) []dankkeyring.SecretRef {
	keys := []string{
		google.SecretKeyApp, google.SecretKeyToken,
		microsoft.SecretKeyApp, microsoft.SecretKeyToken,
		caldav.SecretKeyPassword,
		ical.SecretKeyPassword,
	}

	refs := make([]dankkeyring.SecretRef, 0, len(accounts)*len(keys))
	for _, a := range accounts {
		for _, k := range keys {
			refs = append(refs, dankkeyring.SecretRef{AccountID: a.ID, Key: k})
		}
	}
	return refs
}
