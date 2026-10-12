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
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type quotaRetryHint struct{}

func (quotaRetryHint) Error() string             { return "quota deferred" }
func (quotaRetryHint) RetryAfter() time.Duration { return time.Hour }

func TestDeferredQuotaSchedulesAccountWithoutBlockingOthers(t *testing.T) {
	for _, phase := range []string{"discovery", "calendar"} {
		t.Run(phase, func(t *testing.T) {
			ctx := context.Background()
			client, err := repo.OpenMemory(ctx)
			require.NoError(t, err)
			r := repo.New(client)
			t.Cleanup(func() { _ = r.Close() })
			limited, err := r.CreateAccount(ctx, repo.CreateAccountInput{ID: "a", Kind: account.KindLocal, DisplayName: "Limited"})
			require.NoError(t, err)
			healthy, err := r.CreateAccount(ctx, repo.CreateAccountInput{ID: "b", Kind: account.KindLocal, DisplayName: "Healthy"})
			require.NoError(t, err)
			first := mocks.NewMockProvider(t)
			second := mocks.NewMockProvider(t)
			hint := fmt.Errorf("wrapped Google response: %w", quotaRetryHint{})
			if phase == "discovery" {
				first.EXPECT().ListCalendars(mock.Anything).Return(nil, hint).Once()
			} else {
				first.EXPECT().ListCalendars(mock.Anything).Return([]calendar.Calendar{{RemoteID: "cal", Name: "Cal"}}, nil).Once()
				first.EXPECT().Sync(mock.Anything, mock.Anything, mock.Anything).Return(nil, hint).Once()
			}
			first.EXPECT().Close().Return(nil).Once()
			second.EXPECT().ListCalendars(mock.Anything).Return(nil, nil).Once()
			second.EXPECT().Close().Return(nil).Once()
			factory := mocks.NewMockProviderFactory(t)
			factory.EXPECT().Kind().Return(calendar.AccountLocal)
			factory.EXPECT().Build(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, a calendar.Account, _ calendar.SecretStore) (calendar.Provider, error) {
				if a.ID == limited.ID {
					return first, nil
				}
				return second, nil
			})
			registry := calendar.NewRegistry()
			registry.Register(factory)
			now := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
			e := NewEngine(r, registry, nil, time.Minute)
			e.now = func() time.Time { return now }
			e.runDue(ctx)
			require.False(t, e.due(limited.ID, now.Add(59*time.Minute)))
			require.True(t, e.due(limited.ID, now.Add(time.Hour)))
			require.True(t, e.due(healthy.ID, now.Add(time.Minute)))
			stored, err := r.GetAccount(ctx, limited.ID)
			require.NoError(t, err)
			require.False(t, stored.NeedsReauth)
		})
	}
}
