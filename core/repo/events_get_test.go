package repo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func TestGetEventByUID(t *testing.T) {
	r, ctx := newTestRepo(t)

	_, err := r.CreateAccount(ctx, repo.CreateAccountInput{
		ID:          "acc",
		Kind:        account.KindLocal,
		DisplayName: "Acc",
	})
	require.NoError(t, err)

	cal, err := r.UpsertCalendar(ctx, repo.UpsertCalendarInput{
		AccountID: "acc",
		RemoteID:  "dir:main",
		Name:      "Main",
	})
	require.NoError(t, err)

	// A row far outside any window the GUI keeps loaded still resolves by UID.
	start := time.Date(2031, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = r.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID: cal.ID,
		UID:        "nye-2031",
		Summary:    "New Year's Eve",
		Start:      start,
		End:        start.Add(24 * time.Hour),
	})
	require.NoError(t, err)

	got, err := r.GetEventByUID(ctx, "nye-2031", "")
	require.NoError(t, err)
	require.Equal(t, "New Year's Eve", got.Summary)
	require.Equal(t, cal.ID, got.Edges.Calendar.ID)

	got, err = r.GetEventByUID(ctx, "nye-2031", cal.ID)
	require.NoError(t, err)
	require.Equal(t, "New Year's Eve", got.Summary)

	_, err = r.GetEventByUID(ctx, "nye-2031", "other-cal")
	require.True(t, repo.IsNotFound(err))

	_, err = r.GetEventByUID(ctx, "missing", "")
	require.True(t, repo.IsNotFound(err))
}
