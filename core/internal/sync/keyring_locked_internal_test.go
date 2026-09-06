package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/keyring"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func TestLockedKeyringPollsInsteadOfFlaggingReauth(t *testing.T) {
	ctx := context.Background()
	client, err := repo.OpenMemory(ctx)
	require.NoError(t, err)
	r := repo.New(client)
	t.Cleanup(func() { _ = r.Close() })

	acc, err := r.CreateAccount(ctx, repo.CreateAccountInput{ID: "acc", Kind: account.KindGoogle, DisplayName: "Acc"})
	require.NoError(t, err)

	lockedErr := fmt.Errorf("missing google app credentials: %w", keyring.ErrLocked)
	provider := mocks.NewMockProvider(t)
	provider.EXPECT().ListCalendars(mock.Anything).Return(nil, nil).Maybe()
	provider.EXPECT().Close().Return(nil).Maybe()

	factory := mocks.NewMockProviderFactory(t)
	factory.EXPECT().Kind().Return(calendar.AccountGoogle)
	builds := 0
	factory.EXPECT().Build(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(context.Context, calendar.Account, calendar.SecretStore) (calendar.Provider, error) {
		builds++
		if builds <= 2 {
			return nil, lockedErr
		}
		return provider, nil
	})
	registry := calendar.NewRegistry()
	registry.Register(factory)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := NewEngine(r, registry, nil, 30*time.Minute)
	e.now = func() time.Time { return now }
	var published []string
	e.SetNotifier(func(topic string, _ any) { published = append(published, topic) })

	require.ErrorIs(t, e.SyncAccount(ctx, acc), keyring.ErrLocked)
	stored, err := r.GetAccount(ctx, acc.ID)
	require.NoError(t, err)
	require.False(t, stored.NeedsReauth, "a locked keyring is not a dead credential")
	require.False(t, e.due(acc.ID, now.Add(keyringLockedRetry-time.Second)))
	require.True(t, e.due(acc.ID, now.Add(keyringLockedRetry+time.Second)))
	require.Equal(t, []string{"accounts"}, published)

	require.ErrorIs(t, e.SyncAccount(ctx, acc), keyring.ErrLocked)
	require.Equal(t, []string{"accounts"}, published, "staying locked must not republish")

	require.NoError(t, e.SyncAccount(ctx, acc))
	require.False(t, e.due(acc.ID, now.Add(keyringLockedRetry+time.Second)), "unlocked account returns to the base interval")
	require.True(t, e.due(acc.ID, now.Add(31*time.Minute)))
	require.Equal(t, "accounts", published[len(published)-1])
	require.Len(t, published, 3)
}
