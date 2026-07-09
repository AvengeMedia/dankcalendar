package microsoft

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

func recEvent(rules ...string) *cal.Event {
	return &cal.Event{
		Start:      time.Date(2026, 7, 8, 14, 0, 0, 0, time.UTC), // a Wednesday
		End:        time.Date(2026, 7, 8, 15, 0, 0, 0, time.UTC),
		Recurrence: &cal.Recurrence{RRule: rules},
	}
}

func TestRecurrenceToGraph(t *testing.T) {
	tests := []struct {
		name    string
		event   *cal.Event
		pattern *graphRecurrencePattern
		rng     *graphRecurrenceRange
	}{
		{
			name:    "daily",
			event:   recEvent("FREQ=DAILY"),
			pattern: &graphRecurrencePattern{Type: "daily", Interval: 1},
			rng:     &graphRecurrenceRange{Type: "noEnd", StartDate: "2026-07-08", RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "weekly with byday",
			event:   recEvent("FREQ=WEEKLY;BYDAY=MO,WE"),
			pattern: &graphRecurrencePattern{Type: "weekly", Interval: 1, DaysOfWeek: []string{"monday", "wednesday"}},
			rng:     &graphRecurrenceRange{Type: "noEnd", StartDate: "2026-07-08", RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "weekly without byday defaults to start weekday",
			event:   recEvent("FREQ=WEEKLY"),
			pattern: &graphRecurrencePattern{Type: "weekly", Interval: 1, DaysOfWeek: []string{"wednesday"}},
			rng:     &graphRecurrenceRange{Type: "noEnd", StartDate: "2026-07-08", RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "monthly uses day of month",
			event:   recEvent("FREQ=MONTHLY;INTERVAL=2"),
			pattern: &graphRecurrencePattern{Type: "absoluteMonthly", Interval: 2, DayOfMonth: 8},
			rng:     &graphRecurrenceRange{Type: "noEnd", StartDate: "2026-07-08", RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "yearly uses day and month",
			event:   recEvent("FREQ=YEARLY"),
			pattern: &graphRecurrencePattern{Type: "absoluteYearly", Interval: 1, DayOfMonth: 8, Month: 7},
			rng:     &graphRecurrenceRange{Type: "noEnd", StartDate: "2026-07-08", RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "count",
			event:   recEvent("FREQ=DAILY;COUNT=10"),
			pattern: &graphRecurrencePattern{Type: "daily", Interval: 1},
			rng:     &graphRecurrenceRange{Type: "numbered", StartDate: "2026-07-08", NumberOfOccurrences: 10, RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "until date only",
			event:   recEvent("FREQ=DAILY;UNTIL=20261231"),
			pattern: &graphRecurrencePattern{Type: "daily", Interval: 1},
			rng:     &graphRecurrenceRange{Type: "endDate", StartDate: "2026-07-08", EndDate: "2026-12-31", RecurrenceTimeZone: "UTC"},
		},
		{
			name:    "until with utc time",
			event:   recEvent("FREQ=WEEKLY;BYDAY=WE;UNTIL=20261231T235959Z"),
			pattern: &graphRecurrencePattern{Type: "weekly", Interval: 1, DaysOfWeek: []string{"wednesday"}},
			rng:     &graphRecurrenceRange{Type: "endDate", StartDate: "2026-07-08", EndDate: "2026-12-31", RecurrenceTimeZone: "UTC"},
		},
		{name: "no recurrence", event: &cal.Event{Start: time.Now()}},
		{name: "multiple rules unmappable", event: recEvent("FREQ=DAILY", "FREQ=WEEKLY")},
		{name: "sub-daily freq unmappable", event: recEvent("FREQ=HOURLY")},
		{name: "ordinal byday unmappable", event: recEvent("FREQ=WEEKLY;BYDAY=2MO")},
		{name: "byday outside weekly unmappable", event: recEvent("FREQ=MONTHLY;BYDAY=MO")},
		{name: "bymonthday unmappable", event: recEvent("FREQ=MONTHLY;BYMONTHDAY=15")},
		{name: "bad until unmappable", event: recEvent("FREQ=DAILY;UNTIL=someday")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recurrenceToGraph(tt.event)
			if tt.pattern == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.pattern, got.Pattern)
			assert.Equal(t, *tt.rng, got.Range)
		})
	}
}

func TestEventToGraphReminders(t *testing.T) {
	base := recEvent("FREQ=DAILY")

	t.Run("earliest popup wins", func(t *testing.T) {
		ev := *base
		ev.Reminders = []cal.Reminder{
			{Method: "popup", Minutes: 0},
			{Method: "email", Minutes: 120},
			{Method: "popup", Minutes: 10},
		}
		g := eventToGraph(&ev)
		require.NotNil(t, g.IsReminderOn)
		assert.True(t, *g.IsReminderOn)
		require.NotNil(t, g.ReminderMinutesBeforeStart)
		assert.Equal(t, 10, *g.ReminderMinutesBeforeStart)
	})

	t.Run("no popup reminders turns reminder off", func(t *testing.T) {
		ev := *base
		ev.Reminders = []cal.Reminder{{Method: "email", Minutes: 30}}
		g := eventToGraph(&ev)
		require.NotNil(t, g.IsReminderOn)
		assert.False(t, *g.IsReminderOn)
		assert.Nil(t, g.ReminderMinutesBeforeStart)
	})

	t.Run("recurrence attached", func(t *testing.T) {
		g := eventToGraph(base)
		require.NotNil(t, g.Recurrence)
		assert.Equal(t, "daily", g.Recurrence.Pattern.Type)
	})
}

func TestGraphToEventReminderRoundTrip(t *testing.T) {
	on, minutes := true, 15
	ev := graphToEvent(graphEvent{
		ID:                         "id1",
		IsReminderOn:               &on,
		ReminderMinutesBeforeStart: &minutes,
	})
	assert.Equal(t, []cal.Reminder{{Method: "popup", Minutes: 15}}, ev.Reminders)

	off := false
	ev = graphToEvent(graphEvent{ID: "id2", IsReminderOn: &off, ReminderMinutesBeforeStart: &minutes})
	assert.Empty(t, ev.Reminders)
}
