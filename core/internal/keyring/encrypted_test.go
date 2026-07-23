package keyring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type memStore struct {
	m map[string][]byte
}

func (s *memStore) Get(_ context.Context, accountID, key string) ([]byte, error) {
	v, ok := s.m[accountID+"/"+key]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), v...), nil
}

func (s *memStore) Set(_ context.Context, accountID, key string, value []byte) error {
	if s.m == nil {
		s.m = map[string][]byte{}
	}
	s.m[accountID+"/"+key] = append([]byte(nil), value...)
	return nil
}

func (s *memStore) Delete(_ context.Context, accountID, key string) error {
	delete(s.m, accountID+"/"+key)
	return nil
}

func TestEncryptedSecretStoreRoundTrip(t *testing.T) {
	enc, err := NewEncryptedSecretStore(&memStore{}, []byte("portal-master-secret"))
	require.NoError(t, err)

	ctx := context.Background()
	plain := []byte("oauth-refresh-token")
	require.NoError(t, enc.Set(ctx, "acc1", "refresh", plain))

	got, err := enc.Get(ctx, "acc1", "refresh")
	require.NoError(t, err)
	require.Equal(t, plain, got)

	// Stored form must not be plaintext.
	raw, err := enc.inner.Get(ctx, "acc1", "refresh")
	require.NoError(t, err)
	require.NotEqual(t, plain, raw)
	require.True(t, len(raw) > len(encPrefix))
}

func TestEncryptedSecretStorePlaintextLegacy(t *testing.T) {
	inner := &memStore{m: map[string][]byte{"acc1/refresh": []byte("legacy-plain")}}
	enc, err := NewEncryptedSecretStore(inner, []byte("portal-master-secret"))
	require.NoError(t, err)

	got, err := enc.Get(context.Background(), "acc1", "refresh")
	require.NoError(t, err)
	require.Equal(t, []byte("legacy-plain"), got)
}
