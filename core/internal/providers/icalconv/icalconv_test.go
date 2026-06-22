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

	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
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

func TestEventFromComponentCapturesTZID(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//test//test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:tz-1\r\nSUMMARY:Standup\r\n" +
		"DTSTART;TZID=Australia/Sydney:20250906T090000\r\n" +
		"DTEND;TZID=Australia/Sydney:20250906T093000\r\n" +
		"RRULE:FREQ=MONTHLY;BYDAY=3SA\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"

	doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)
	require.Len(t, doc.Events(), 1)

	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
	require.True(t, ok)

	assert.Equal(t, "Australia/Sydney", got.StartTimeZone)
	assert.Equal(t, "Australia/Sydney", got.EndTimeZone)
	// 09:00 AEST (UTC+10, before DST starts in October) is 23:00 UTC the day before.
	assert.Equal(t, time.Date(2025, 9, 5, 23, 0, 0, 0, time.UTC), got.Start.UTC())
}

// Forwarded invites often label DTSTART with a non-IANA TZID. go-ical can't
// resolve those and would leave a zero time (the "Jan 01, 0001" in #17); the
// resolver must recover a real instant from a Windows name, an embedded
// VTIMEZONE, or at worst the literal wall-clock.
func TestEventFromComponentResolvesForwardedZones(t *testing.T) {
	parse := func(t *testing.T, body string) cal.Event {
		t.Helper()
		raw := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//x//x//EN\r\n" + body + "END:VCALENDAR\r\n"
		doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
		require.NoError(t, err)
		require.Len(t, doc.Events(), 1)
		got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
		require.True(t, ok)
		assert.False(t, got.Start.IsZero(), "start must not be the zero time")
		return got
	}

	t.Run("windows name", func(t *testing.T) {
		got := parse(t, "BEGIN:VEVENT\r\nUID:w-1\r\nSUMMARY:s\r\n"+
			"DTSTART;TZID=Eastern Standard Time:20260115T090000\r\n"+
			"DTEND;TZID=Eastern Standard Time:20260115T100000\r\n"+
			"END:VEVENT\r\n")
		// 09:00 EST (UTC-5 in January) is 14:00 UTC, mapped to the IANA zone.
		assert.Equal(t, "America/New_York", got.StartTimeZone)
		assert.Equal(t, time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC), got.Start.UTC())
	})

	t.Run("embedded vtimezone", func(t *testing.T) {
		got := parse(t, "BEGIN:VTIMEZONE\r\nTZID:Customized Time Zone\r\n"+
			"BEGIN:STANDARD\r\nDTSTART:16010101T000000\r\nTZOFFSETFROM:-0500\r\nTZOFFSETTO:-0500\r\n"+
			"END:STANDARD\r\nEND:VTIMEZONE\r\n"+
			"BEGIN:VEVENT\r\nUID:c-1\r\nSUMMARY:s\r\n"+
			"DTSTART;TZID=Customized Time Zone:20260115T090000\r\n"+
			"DTEND;TZID=Customized Time Zone:20260115T100000\r\n"+
			"END:VEVENT\r\n")
		// Fixed -0500 offset from the VTIMEZONE: 09:00 local is 14:00 UTC.
		assert.Equal(t, time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC), got.Start.UTC())
	})

	t.Run("unresolvable keeps wall clock", func(t *testing.T) {
		got := parse(t, "BEGIN:VEVENT\r\nUID:u-1\r\nSUMMARY:s\r\n"+
			"DTSTART;TZID=Totally Made Up:20260115T090000\r\n"+
			"DTEND;TZID=Totally Made Up:20260115T100000\r\n"+
			"END:VEVENT\r\n")
		assert.Equal(t, 2026, got.Start.Year())
		assert.Equal(t, 9, got.Start.UTC().Hour())
	})
}

// A floating DTSTART (no TZID, no Z) is local time per RFC 5545 §3.3.5, not
// UTC. It must be read in the calendar's zone so a 09:00 entry stays 09:00 local
// and recurrence expands on that wall clock.
func TestFloatingUsesCalendarZone(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//x//x//EN\r\nX-WR-TIMEZONE:Australia/Sydney\r\n" +
		"BEGIN:VEVENT\r\nUID:f-1\r\nSUMMARY:s\r\nDTSTART:20260115T090000\r\nDTEND:20260115T100000\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=TH\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)

	// calTZ wins over X-WR-TIMEZONE when both are present.
	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, "Australia/Sydney"))
	require.True(t, ok)
	assert.Equal(t, "Australia/Sydney", got.StartTimeZone)
	// 09:00 AEDT (UTC+11 in January) is 22:00 UTC the previous day.
	assert.Equal(t, time.Date(2026, 1, 14, 22, 0, 0, 0, time.UTC), got.Start.UTC())

	// Falls back to the document's X-WR-TIMEZONE when no calendar zone is given.
	got, ok = EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
	require.True(t, ok)
	assert.Equal(t, "Australia/Sydney", got.StartTimeZone)
}

// An EXDATE carrying a different TZID than the series must still exclude the
// right occurrence: go-ical drops the TZID from the raw value, so the resolver
// normalizes EXDATE/RDATE to absolute UTC instants at parse time.
func TestExdateTZIDNormalizedToUTC(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//x//x//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:x-1\r\nSUMMARY:s\r\n" +
		"DTSTART;TZID=Australia/Sydney:20260117T090000\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=SA\r\n" +
		"EXDATE;TZID=Australia/Sydney:20260124T090000\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"

	doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)

	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
	require.True(t, ok)
	require.NotNil(t, got.Recurrence)
	// 09:00 AEDT on 2026-01-24 is 22:00 UTC on the 23rd.
	require.Len(t, got.Recurrence.ExDate, 1)
	assert.Equal(t, "20260123T220000Z", got.Recurrence.ExDate[0])
}

// A fixed-offset zone defined only by a VTIMEZONE is named with an Etc/GMT zone
// so recurrence can reconstruct the offset after the instant is persisted.
func TestEmbeddedFixedOffsetGetsEtcZone(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//x//x//EN\r\n" +
		"BEGIN:VTIMEZONE\r\nTZID:Customized Time Zone\r\n" +
		"BEGIN:STANDARD\r\nDTSTART:16010101T000000\r\nTZOFFSETFROM:+1000\r\nTZOFFSETTO:+1000\r\n" +
		"END:STANDARD\r\nEND:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\nUID:c-2\r\nSUMMARY:s\r\n" +
		"DTSTART;TZID=Customized Time Zone:20260115T090000\r\n" +
		"RRULE:FREQ=WEEKLY\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)

	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
	require.True(t, ok)
	assert.Equal(t, "Etc/GMT-10", got.StartTimeZone)
	assert.Equal(t, time.Date(2026, 1, 14, 23, 0, 0, 0, time.UTC), got.Start.UTC())
}

// The real #17 payload: a Microsoft Exchange invite whose TZID is the Windows
// name "Eastern Standard Time". go-ical cannot load that name, so the start used
// to render as 0001-01-01. The event is in June (daylight time), so the label
// "Standard" is a red herring -- it must resolve to EDT (UTC-4), not EST.
func TestForwardedExchangeWindowsZone(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nMETHOD:REQUEST\r\nPRODID:Microsoft Exchange Server 2010\r\nVERSION:2.0\r\n" +
		"BEGIN:VTIMEZONE\r\nTZID:Eastern Standard Time\r\n" +
		"BEGIN:STANDARD\r\nDTSTART:16010101T020000\r\nTZOFFSETFROM:-0400\r\nTZOFFSETTO:-0500\r\nRRULE:FREQ=YEARLY;INTERVAL=1;BYDAY=1SU;BYMONTH=11\r\nEND:STANDARD\r\n" +
		"BEGIN:DAYLIGHT\r\nDTSTART:16010101T020000\r\nTZOFFSETFROM:-0500\r\nTZOFFSETTO:-0400\r\nRRULE:FREQ=YEARLY;INTERVAL=1;BYDAY=2SU;BYMONTH=3\r\nEND:DAYLIGHT\r\n" +
		"END:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\nUID:exchange-1\r\nSUMMARY:Testing CalDav Forwarding Dt/Tm\r\n" +
		"DTSTART;TZID=Eastern Standard Time:20260618T070000\r\n" +
		"DTEND;TZID=Eastern Standard Time:20260618T073000\r\n" +
		"DTSTAMP:20260618T105420Z\r\n" +
		"BEGIN:VALARM\r\nDESCRIPTION:REMINDER\r\nTRIGGER;RELATED=START:P\r\nACTION:DISPLAY\r\nEND:VALARM\r\n" +
		"END:VEVENT\r\nEND:VCALENDAR\r\n"

	doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)

	got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
	require.True(t, ok)

	assert.False(t, got.Start.IsZero(), "start must not be the zero time")
	assert.Equal(t, "America/New_York", got.StartTimeZone)
	// 07:00 EDT (UTC-4 in June) is 11:00 UTC; the description says it should read 07:00.
	assert.Equal(t, time.Date(2026, 6, 18, 11, 0, 0, 0, time.UTC), got.Start.UTC())
	assert.Equal(t, time.Date(2026, 6, 18, 11, 30, 0, 0, time.UTC), got.End.UTC())
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

func TestMeetingURLExtraction(t *testing.T) {
	const head = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//test//test//EN\r\nBEGIN:VEVENT\r\nUID:m-1\r\nSUMMARY:Meeting\r\n"
	const tail = "END:VEVENT\r\nEND:VCALENDAR\r\n"

	tests := []struct {
		name  string
		props string
		want  string
	}{
		{
			name:  "google conference",
			props: "X-GOOGLE-CONFERENCE:https://meet.google.com/abc-defg-hij\r\n",
			want:  "https://meet.google.com/abc-defg-hij",
		},
		{
			name:  "microsoft teams property",
			props: "X-MICROSOFT-SKYPETEAMSMEETINGURL:https://teams.microsoft.com/l/meetup-join/19%3aabc\r\n",
			want:  "https://teams.microsoft.com/l/meetup-join/19%3aabc",
		},
		{
			name: "conference prop prefers video over audio dial-in",
			props: "CONFERENCE;VALUE=URI;FEATURE=AUDIO:tel:+1-555-0100,,123456\r\n" +
				"CONFERENCE;VALUE=URI;FEATURE=VIDEO:https://chat.example.com/video?id=42\r\n",
			want: "https://chat.example.com/video?id=42",
		},
		{
			name:  "zoom link recovered from description",
			props: "DESCRIPTION:Join here https://us02web.zoom.us/j/87654321 see you\r\n",
			want:  "https://us02web.zoom.us/j/87654321",
		},
		{
			name:  "no meeting",
			props: "DESCRIPTION:Just a regular event\r\n",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := ical.NewDecoder(strings.NewReader(head + tc.props + tail)).Decode()
			require.NoError(t, err)
			require.Len(t, doc.Events(), 1)

			got, ok := EventFromComponent("cal-1", doc.Events()[0].Component, NewTZResolver(doc, ""))
			require.True(t, ok)
			assert.Equal(t, tc.want, got.MeetingURL)
		})
	}
}

func TestMeetingURLRoundTrip(t *testing.T) {
	src := &cal.Event{
		Summary:    "Sprint review",
		MeetingURL: "https://meet.google.com/xyz-abcd-efg",
		Start:      time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC),
		End:        time.Date(2026, 5, 7, 15, 0, 0, 0, time.UTC),
	}
	got := roundTrip(t, src, "rt-meeting")
	assert.Equal(t, src.MeetingURL, got.MeetingURL)
}

func TestTriggerMinutesRoundTrip(t *testing.T) {
	for _, minutes := range []int{0, 5, 10, 60, 1440, -30} {
		got, ok := triggerMinutes(triggerFromMinutes(minutes))
		require.True(t, ok, "minutes=%d", minutes)
		assert.Equal(t, minutes, got, "minutes=%d", minutes)
	}
}
