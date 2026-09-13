package icsimport

import (
	"context"
	"testing"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflicts(t *testing.T) {
	ctx := context.Background()
	client, err := repo.OpenMemory(ctx)
	require.NoError(t, err)
	r := repo.New(client)
	t.Cleanup(func() { _ = r.Close() })
	_, err = r.CreateAccount(ctx, repo.CreateAccountInput{ID: "acc", Kind: account.KindLocal, DisplayName: "Local"})
	require.NoError(t, err)
	_, err = r.UpsertCalendar(ctx, repo.UpsertCalendarInput{ID: "cal", AccountID: "acc", RemoteID: "remote", Name: "Work"})
	require.NoError(t, err)
	start := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		uid          string
		start, end   time.Time
		transparency string
		rec          map[string]any
	}{
		{"overlap", start.Add(30 * time.Minute), start.Add(2 * time.Hour), "", nil},
		{"adjacent", start.Add(time.Hour), start.Add(2 * time.Hour), "", nil},
		{"free", start, start.Add(time.Hour), "TRANSPARENT", nil},
		{"same", start, start.Add(time.Hour), "", nil},
		{"recurring", start.AddDate(0, 0, -5).Add(-30 * time.Minute), start.AddDate(0, 0, -5).Add(30 * time.Minute), "", map[string]any{"rrule": []string{"FREQ=DAILY"}}},
		{"tomorrow", start.AddDate(0, 0, 1), start.AddDate(0, 0, 1).Add(time.Hour), "", nil},
	} {
		_, err := r.UpsertEvent(ctx, repo.UpsertEventInput{CalendarID: "cal", UID: tc.uid, Start: tc.start, End: tc.end, Transparency: tc.transparency, Recurrence: tc.rec})
		require.NoError(t, err)
	}
	ev := calendar.Event{UID: "same", Start: start, End: start.Add(time.Hour)}
	conflicts, err := Conflicts(ctx, r, ev)
	require.NoError(t, err)
	var uids []string
	for _, conflict := range conflicts {
		uids = append(uids, conflict.UID)
	}
	assert.ElementsMatch(t, []string{"overlap", "recurring"}, uids)
	ev.Transparency = "TRANSPARENT"
	conflicts, err = Conflicts(ctx, r, ev)
	require.NoError(t, err)
	assert.Empty(t, conflicts)
	ev.Transparency = ""
	ev.Recurrence = &calendar.Recurrence{RRule: []string{"FREQ=DAILY;COUNT=2"}}
	conflicts, err = Conflicts(ctx, r, ev)
	require.NoError(t, err)
	uids = nil
	for _, conflict := range conflicts {
		uids = append(uids, conflict.UID)
	}
	assert.Contains(t, uids, "tomorrow")
}
