package repo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func TestListEventsExpandsRecurring(t *testing.T) {
	r, ctx := newTestRepo(t)

	_, err := r.CreateAccount(ctx, repo.CreateAccountInput{
		ID:          "acc",
		Kind:        account.KindLocal,
		DisplayName: "Acc",
	})
	require.NoError(t, err)

	cal, err := r.UpsertCalendar(ctx, repo.UpsertCalendarInput{
		AccountID: "acc",
		RemoteID:  "file:cal.ics",
		Name:      "Cal",
	})
	require.NoError(t, err)

	// Yearly birthday whose master starts decades before the query window.
	_, err = r.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID: cal.ID,
		UID:        "bday",
		Summary:    "Birthday",
		Start:      time.Date(1992, 6, 9, 0, 0, 0, 0, time.UTC),
		End:        time.Date(1992, 6, 10, 0, 0, 0, 0, time.UTC),
		AllDay:     true,
		Recurrence: map[string]any{"rrule": []string{"FREQ=YEARLY;BYMONTH=6;BYMONTHDAY=9"}},
	})
	require.NoError(t, err)

	// Weekly meeting; one instance moved, one cancelled.
	weeklyStart := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
	_, err = r.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID: cal.ID,
		UID:        "weekly",
		Summary:    "Weekly",
		Start:      weeklyStart,
		End:        weeklyStart.Add(time.Hour),
		Recurrence: map[string]any{"rrule": []string{"FREQ=WEEKLY;BYDAY=MO"}},
	})
	require.NoError(t, err)

	moved := time.Date(2026, 6, 8, 14, 0, 0, 0, time.UTC)
	_, err = r.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID:    cal.ID,
		UID:           "weekly-moved",
		Summary:       "Weekly (moved)",
		Start:         moved.Add(2 * time.Hour),
		End:           moved.Add(3 * time.Hour),
		RecurringID:   "weekly",
		OriginalStart: moved,
	})
	require.NoError(t, err)

	cancelled := time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC)
	_, err = r.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID:    cal.ID,
		UID:           "weekly-cancelled",
		Summary:       "",
		Status:        event.StatusCancelled,
		Start:         cancelled,
		End:           cancelled.Add(time.Hour),
		RecurringID:   "weekly",
		OriginalStart: cancelled,
	})
	require.NoError(t, err)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	events, total, err := r.ListEvents(ctx, repo.ListEventsParams{
		Filter: repo.EventFilter{From: &from, To: &to, IncludeRecurring: true},
	})
	require.NoError(t, err)
	require.Equal(t, len(events), total)

	starts := map[string][]string{}
	for _, e := range events {
		starts[e.Summary] = append(starts[e.Summary], e.Start.UTC().Format("01-02 15:04"))
	}

	require.Equal(t, []string{"06-09 00:00"}, starts["Birthday"])
	// Mondays in June minus the moved (8th) and cancelled (15th) instances.
	require.Equal(t, []string{"06-01 14:00", "06-22 14:00", "06-29 14:00"}, starts["Weekly"])
	require.Equal(t, []string{"06-08 16:00"}, starts["Weekly (moved)"])
	require.Empty(t, starts[""])

	// The occurrence matching the master start keeps its row id; generated
	// occurrences carry no id so the UI treats them as read-only.
	for _, e := range events {
		switch {
		case e.Summary == "Weekly" && e.Start.Equal(weeklyStart):
			require.NotEmpty(t, e.ID)
		case e.Summary == "Weekly":
			require.Empty(t, e.ID)
		}
	}
}

func TestListEventsWithoutWindowSkipsExpansion(t *testing.T) {
	r, ctx := newTestRepo(t)

	_, err := r.CreateAccount(ctx, repo.CreateAccountInput{ID: "acc", Kind: account.KindLocal, DisplayName: "Acc"})
	require.NoError(t, err)
	cal, err := r.UpsertCalendar(ctx, repo.UpsertCalendarInput{AccountID: "acc", RemoteID: "file:c.ics", Name: "C"})
	require.NoError(t, err)

	_, err = r.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID: cal.ID,
		UID:        "weekly",
		Summary:    "Weekly",
		Start:      time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC),
		End:        time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC),
		Recurrence: map[string]any{"rrule": []string{"FREQ=WEEKLY"}},
	})
	require.NoError(t, err)

	events, total, err := r.ListEvents(ctx, repo.ListEventsParams{
		Filter: repo.EventFilter{IncludeRecurring: true},
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, events, 1)
	require.Equal(t, "weekly", events[0].UID)
}
