package icalconv

import (
	"bytes"
	"strings"
	"testing"
	"time"

	ical "github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

// roundTrip encodes an event to ICS and parses it back, exercising both
// directions of the mapping.
func roundTrip(t *testing.T, ev *cal.Event, uid string) cal.Event {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, ical.NewEncoder(&buf).Encode(CalendarFromEvent(ev, uid)))

	doc, err := ical.NewDecoder(strings.NewReader(buf.String())).Decode()
	require.NoError(t, err)
	require.Len(t, doc.Events(), 1)

	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component)
	require.True(t, ok)
	return got
}

func TestRoundTripCoreFields(t *testing.T) {
	src := &cal.Event{
		Summary:     "Sync up",
		Description: "Weekly team sync",
		Location:    "Hall B",
		URL:         "https://example.com/sync",
		Status:      cal.EventTentative,
		Start:       time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 5, 7, 15, 0, 0, 0, time.UTC),
		Recurrence:  &cal.Recurrence{RRule: []string{"FREQ=WEEKLY;BYDAY=TH"}},
		Organizer:   &cal.Attendee{Email: "alice@example.com", DisplayName: "Alice", Organizer: true},
		Attendees: []cal.Attendee{
			{Email: "bob@example.com", DisplayName: "Bob", Role: "REQ-PARTICIPANT", Status: "accepted"},
		},
	}

	got := roundTrip(t, src, "uid-9")
	assert.Equal(t, "uid-9", got.UID)
	assert.Equal(t, "cal-1", got.CalendarID)
	assert.Equal(t, src.Summary, got.Summary)
	assert.Equal(t, src.Description, got.Description)
	assert.Equal(t, src.Location, got.Location)
	assert.Equal(t, src.URL, got.URL)
	assert.Equal(t, src.Status, got.Status)
	assert.Equal(t, src.Start, got.Start.UTC())
	assert.Equal(t, src.End, got.End.UTC())

	require.NotNil(t, got.Recurrence)
	assert.Equal(t, src.Recurrence.RRule, got.Recurrence.RRule)
	require.NotNil(t, got.Organizer)
	assert.Equal(t, "alice@example.com", got.Organizer.Email)
	require.Len(t, got.Attendees, 1)
	assert.Equal(t, "bob@example.com", got.Attendees[0].Email)
	assert.Equal(t, "accepted", got.Attendees[0].Status)
}

func TestRoundTripAllDay(t *testing.T) {
	src := &cal.Event{
		Summary: "Conference",
		AllDay:  true,
		Start:   time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
		End:     time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
	}

	got := roundTrip(t, src, "day-9")
	assert.True(t, got.AllDay)
	assert.Equal(t, got.Start.Add(24*time.Hour), got.End, "zero-length all-day event should expand to one day")
}

func TestRoundTripReminders(t *testing.T) {
	src := &cal.Event{
		Summary:   "Standup",
		Start:     time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC),
		End:       time.Date(2026, 5, 7, 9, 15, 0, 0, time.UTC),
		Reminders: []cal.Reminder{{Method: "popup", Minutes: 10}, {Method: "email", Minutes: 60}},
	}

	got := roundTrip(t, src, "rem-1")
	// Email reminders are dropped on write; only the popup survives.
	require.Len(t, got.Reminders, 1)
	assert.Equal(t, "popup", got.Reminders[0].Method)
	assert.Equal(t, 10, got.Reminders[0].Minutes)
}

func TestStripMailto(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"mailto:a@example.com", "a@example.com"},
		{"MAILTO:a@example.com", "a@example.com"},
		{"a@example.com", "a@example.com"},
		{"", ""},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, StripMailto(tc.raw))
	}
}

func TestTriggerMinutesRoundTrip(t *testing.T) {
	for _, minutes := range []int{0, 5, 10, 60, 1440, -30} {
		got, ok := triggerMinutes(triggerFromMinutes(minutes))
		require.True(t, ok, "minutes=%d", minutes)
		assert.Equal(t, minutes, got, "minutes=%d", minutes)
	}
}
