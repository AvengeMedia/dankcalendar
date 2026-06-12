package calendar_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
)

func TestRegistryDispatchesByKind(t *testing.T) {
	registry := calendar.NewRegistry()

	factory := mocks.NewMockProviderFactory(t)
	provider := mocks.NewMockProvider(t)

	factory.EXPECT().Kind().Return(calendar.AccountLocal)
	factory.EXPECT().
		Build(context.Background(), calendar.Account{ID: "a", Kind: calendar.AccountLocal}, nil).
		Return(provider, nil)

	registry.Register(factory)

	got, err := registry.Build(context.Background(), calendar.Account{ID: "a", Kind: calendar.AccountLocal}, nil)
	require.NoError(t, err)
	require.Equal(t, provider, got)
}

func TestRegistryReturnsErrorForUnknownKind(t *testing.T) {
	registry := calendar.NewRegistry()
	_, err := registry.Build(context.Background(), calendar.Account{Kind: "wat"}, nil)
	require.Error(t, err)
}

func TestRegistrySurfacesFactoryError(t *testing.T) {
	registry := calendar.NewRegistry()
	factory := mocks.NewMockProviderFactory(t)

	wantErr := errors.New("boom")
	factory.EXPECT().Kind().Return(calendar.AccountGoogle)
	factory.EXPECT().
		Build(context.Background(), calendar.Account{Kind: calendar.AccountGoogle}, nil).
		Return(nil, wantErr)

	registry.Register(factory)

	_, err := registry.Build(context.Background(), calendar.Account{Kind: calendar.AccountGoogle}, nil)
	require.ErrorIs(t, err, wantErr)
}
