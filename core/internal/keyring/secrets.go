package keyring

import (
	"context"
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

type SecretStore struct {
	store    *Store
	fallback calendar.SecretStore
}

func NewSecretStore(store *Store, fallback calendar.SecretStore) *SecretStore {
	return &SecretStore{store: store, fallback: fallback}
}

func (s *SecretStore) Get(ctx context.Context, accountID, key string) ([]byte, error) {
	if !s.store.Available() {
		return s.fallback.Get(ctx, accountID, key)
	}

	value, err := s.store.Get(accountID, key)
	switch {
	case errors.Is(err, ErrNotFound):
		return s.fallback.Get(ctx, accountID, key)
	case err != nil:
		return nil, err
	}
	return value, nil
}

func (s *SecretStore) Set(ctx context.Context, accountID, key string, value []byte) error {
	if !s.store.Available() {
		return s.fallback.Set(ctx, accountID, key, value)
	}
	if err := s.store.Set(accountID, key, value); err != nil {
		return err
	}
	_ = s.fallback.Delete(ctx, accountID, key)
	return nil
}

func (s *SecretStore) Delete(ctx context.Context, accountID, key string) error {
	_ = s.fallback.Delete(ctx, accountID, key)
	if !s.store.Available() {
		return nil
	}
	return s.store.Delete(accountID, key)
}
