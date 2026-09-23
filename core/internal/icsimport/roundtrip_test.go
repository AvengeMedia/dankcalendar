package icsimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportRoundTrip(t *testing.T) {
	for _, tc := range []struct{ name, data, title string }{
		{"ICS invitation", invitation, "Quarterly planning"},
		{"VCS event", legacyCalendar, "Café planning"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			path := filepath.Join(dir, "personal.ics")
			require.NoError(t, os.WriteFile(path, []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:tests\r\nEND:VCALENDAR\r\n"), 0600))
			provider, err := local.New(calendar.Account{ID: "local"}, dir)
			require.NoError(t, err)
			defer provider.Close()
			cals, err := provider.ListCalendars(ctx)
			require.NoError(t, err)
			require.Len(t, cals, 1)
			doc, err := Parse([]byte(tc.data))
			require.NoError(t, err)
			_, err = Create(ctx, provider, cals[0], &doc.Events[0])
			require.NoError(t, err)
			events, err := provider.ListEvents(ctx, cals[0], calendar.ListEventsOptions{})
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tc.title, events[0].Summary)
			assert.Equal(t, doc.Events[0].UID, events[0].UID)
			assert.True(t, doc.Events[0].Start.Equal(events[0].Start))
			assert.True(t, doc.Events[0].End.Equal(events[0].End))
			assert.Equal(t, doc.Events[0].Recurrence, events[0].Recurrence)
		})
	}
}
