package google

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

// Google rejects recurring events whose start/end lack an explicit time zone
// ("Missing time zone definition for start time").
func TestToGoogleEventRecurringSetsTimeZone(t *testing.T) {
	ev := &cal.Event{
		Summary:    "standup",
		Start:      time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC),
		End:        time.Date(2026, 7, 10, 16, 30, 0, 0, time.UTC),
		Recurrence: &cal.Recurrence{RRule: []string{"FREQ=WEEKLY;BYDAY=FR"}},
	}

	out := toGoogleEvent(ev)
	require.NotNil(t, out.Start)
	require.NotNil(t, out.End)
	assert.Equal(t, "UTC", out.Start.TimeZone)
	assert.Equal(t, "UTC", out.End.TimeZone)
	assert.Equal(t, []string{"RRULE:FREQ=WEEKLY;BYDAY=FR"}, out.Recurrence)
}

func TestToGoogleEventNonRecurringLeavesTimeZoneEmpty(t *testing.T) {
	ev := &cal.Event{
		Summary: "one-off",
		Start:   time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC),
		End:     time.Date(2026, 7, 10, 16, 30, 0, 0, time.UTC),
	}

	out := toGoogleEvent(ev)
	require.NotNil(t, out.Start)
	assert.Empty(t, out.Start.TimeZone)
	assert.Empty(t, out.Recurrence)
}

func TestToGoogleEventRecurringExistingZoneKept(t *testing.T) {
	ev := &cal.Event{
		Summary:       "standup",
		Start:         time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC),
		End:           time.Date(2026, 7, 10, 16, 30, 0, 0, time.UTC),
		StartTimeZone: "America/New_York",
		EndTimeZone:   "America/New_York",
		Recurrence:    &cal.Recurrence{RRule: []string{"FREQ=DAILY"}},
	}

	out := toGoogleEvent(ev)
	assert.Equal(t, "America/New_York", out.Start.TimeZone)
	assert.Equal(t, "America/New_York", out.End.TimeZone)
}
