package local_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/local"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//dankcal-test//EN
BEGIN:VEVENT
UID:demo-1@dankcal
DTSTAMP:20260506T120000Z
DTSTART:20260507T140000Z
DTEND:20260507T150000Z
SUMMARY:Lunch with Alice
LOCATION:Cafe Nord
END:VEVENT
END:VCALENDAR
`

func TestLocalProviderListsCalendarsAndEvents(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personal.ics"), []byte(sampleICS), 0o600))

	provider, err := local.New(calendar.Account{ID: "test"}, dir)
	require.NoError(t, err)

	ctx := context.Background()
	cals, err := provider.ListCalendars(ctx)
	require.NoError(t, err)
	require.Len(t, cals, 1)
	require.Equal(t, "personal", cals[0].Name)

	events, err := provider.ListEvents(ctx, cals[0], calendar.ListEventsOptions{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "Lunch with Alice", events[0].Summary)
	require.Equal(t, "Cafe Nord", events[0].Location)
	require.True(t, events[0].Start.Before(events[0].End))
}

func TestLocalProviderTimeFiltering(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p.ics"), []byte(sampleICS), 0o600))

	provider, err := local.New(calendar.Account{ID: "t"}, dir)
	require.NoError(t, err)

	ctx := context.Background()
	cals, err := provider.ListCalendars(ctx)
	require.NoError(t, err)

	farFuture := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	events, err := provider.ListEvents(ctx, cals[0], calendar.ListEventsOptions{
		Start: farFuture,
		End:   farFuture.Add(24 * time.Hour),
	})
	require.NoError(t, err)
	require.Empty(t, events)
}
