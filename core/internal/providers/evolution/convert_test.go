package evolution

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// EDS GetObjectList returns bare VEVENT components with no VCALENDAR wrapper and
// no accompanying VTIMEZONE, so these tests pin that conversion path.
func TestEventsFromICS_AllDayRecurring(t *testing.T) {
	obj := "BEGIN:VEVENT\r\n" +
		"UID:4butt8ntu07qicod2u86o8i643@google.com\r\n" +
		"DTSTART;VALUE=DATE:20250720\r\n" +
		"DTEND;VALUE=DATE:20250721\r\n" +
		"RRULE:FREQ=YEARLY\r\n" +
		"SUMMARY:Sahar Anniversary\r\n" +
		"END:VEVENT\r\n"

	events := eventsFromICS("cal-1", []string{obj})

	require.Len(t, events, 1)
	ev := events[0]
	require.Equal(t, "4butt8ntu07qicod2u86o8i643@google.com", ev.UID)
	require.Equal(t, ev.UID, ev.RemoteID)
	require.Equal(t, "cal-1", ev.CalendarID)
	require.Equal(t, "Sahar Anniversary", ev.Summary)
	require.True(t, ev.AllDay)
	require.NotNil(t, ev.Recurrence)
	require.Equal(t, []string{"FREQ=YEARLY"}, ev.Recurrence.RRule)
	require.Equal(t, obj, ev.RawICS)
}

func TestEventsFromICS_ResolvesTZIDWithoutVTIMEZONE(t *testing.T) {
	obj := "BEGIN:VEVENT\r\n" +
		"UID:tz-1\r\n" +
		"DTSTART;TZID=America/New_York:20250720T090000\r\n" +
		"DTEND;TZID=America/New_York:20250720T100000\r\n" +
		"SUMMARY:Meeting\r\n" +
		"END:VEVENT\r\n"

	events := eventsFromICS("cal-1", []string{obj})

	require.Len(t, events, 1)
	ev := events[0]
	require.False(t, ev.AllDay)
	require.Equal(t, "America/New_York", ev.StartTimeZone)
	// 09:00 EDT is 13:00 UTC; proves the IANA TZID resolved without a VTIMEZONE.
	require.Equal(t, 13, ev.Start.UTC().Hour())
}

func TestEventsFromICS_SkipsUndecodable(t *testing.T) {
	require.Empty(t, eventsFromICS("cal-1", []string{"not an ical object"}))
}

func TestRangeQuery(t *testing.T) {
	require.Equal(t, "#t", rangeQuery(time.Time{}, time.Time{}))

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	q := rangeQuery(start, end)
	require.Contains(t, q, "occur-in-time-range?")
	require.Contains(t, q, "20250101T000000Z")
	require.Contains(t, q, "20251231T000000Z")
}
