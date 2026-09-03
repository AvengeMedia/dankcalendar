package icsimport

import (
	"context"
	"strings"
	"testing"
	_ "time/tzdata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
)

const invitation = "BEGIN:VCALENDAR\r\n" +
	"PRODID:-//Example//Mail//EN\r\n" +
	"VERSION:2.0\r\n" +
	"METHOD:REQUEST\r\n" +
	"BEGIN:VTIMEZONE\r\n" +
	"TZID:Europe/Berlin\r\n" +
	"BEGIN:STANDARD\r\n" +
	"DTSTART:19701025T030000\r\n" +
	"RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU\r\n" +
	"TZOFFSETFROM:+0200\r\n" +
	"TZOFFSETTO:+0100\r\n" +
	"END:STANDARD\r\n" +
	"BEGIN:DAYLIGHT\r\n" +
	"DTSTART:19700329T020000\r\n" +
	"RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU\r\n" +
	"TZOFFSETFROM:+0100\r\n" +
	"TZOFFSETTO:+0200\r\n" +
	"END:DAYLIGHT\r\n" +
	"END:VTIMEZONE\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:meeting-42@example.com\r\n" +
	"DTSTAMP:20260901T080000Z\r\n" +
	"DTSTART;TZID=Europe/Berlin:20260910T140000\r\n" +
	"DTEND;TZID=Europe/Berlin:20260910T150000\r\n" +
	"SUMMARY:Quarterly planning\r\n" +
	"LOCATION:Room 4\r\n" +
	"ORGANIZER;CN=Alice:mailto:alice@example.com\r\n" +
	"ATTENDEE;CN=Bob;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE:mailto:b\r\n" +
	" ob@example.com\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func TestParseInvitation(t *testing.T) {
	doc, err := Parse([]byte(invitation))
	require.NoError(t, err)

	assert.Equal(t, "REQUEST", doc.Method)
	require.Len(t, doc.Events, 1)

	ev := doc.Events[0]
	assert.Equal(t, "meeting-42@example.com", ev.UID)
	assert.Equal(t, "Quarterly planning", ev.Summary)
	assert.Equal(t, "Room 4", ev.Location)
	assert.Equal(t, "2026-09-10T12:00:00Z", ev.Start.UTC().Format("2006-01-02T15:04:05Z"))
	assert.Equal(t, "2026-09-10T13:00:00Z", ev.End.UTC().Format("2006-01-02T15:04:05Z"))
	assert.Equal(t, "Europe/Berlin", ev.StartTimeZone)

	require.NotNil(t, ev.Organizer)
	assert.Equal(t, "alice@example.com", ev.Organizer.Email)
	require.Len(t, ev.Attendees, 1)
	assert.Equal(t, "bob@example.com", ev.Attendees[0].Email)
	assert.Equal(t, calendar.ResponseNeedsAction, ev.Attendees[0].Status)
}

func TestParseConcatenatedCalendarsAndUnixNewlines(t *testing.T) {
	second := strings.ReplaceAll(strings.ReplaceAll(invitation, "meeting-42", "meeting-43"), "\r\n", "\n")
	doc, err := Parse([]byte(invitation + "\n" + second))
	require.NoError(t, err)

	require.Len(t, doc.Events, 2)
	assert.Equal(t, "meeting-42@example.com", doc.Events[0].UID)
	assert.Equal(t, "meeting-43@example.com", doc.Events[1].UID)
}

func TestParseSkipsRecurrenceExceptions(t *testing.T) {
	exception := strings.Replace(invitation, "DTSTAMP:", "RECURRENCE-ID;TZID=Europe/Berlin:20260910T140000\r\nDTSTAMP:", 1)

	_, err := Parse([]byte(exception))
	assert.ErrorIs(t, err, ErrNoEvents)

	doc, err := Parse([]byte(invitation + exception))
	require.NoError(t, err)
	require.Len(t, doc.Events, 1)
	assert.Empty(t, doc.Events[0].RecurringID)
}

func TestParseRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"plain text":     "hello world\n",
		"no events":      "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:x\r\nEND:VCALENDAR\r\n",
		"todo only":      "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:x\r\nBEGIN:VTODO\r\nUID:t1\r\nSUMMARY:x\r\nEND:VTODO\r\nEND:VCALENDAR\r\n",
		"event sans uid": "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:x\r\nBEGIN:VEVENT\r\nDTSTART:20260910T120000Z\r\nSUMMARY:x\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		"too large":      strings.Repeat("X", MaxBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			doc, err := Parse([]byte(input))
			assert.Error(t, err)
			assert.Nil(t, doc)
		})
	}
}

type importingProvider struct {
	*mocks.MockProvider
	got *calendar.Event
}

func (p *importingProvider) ImportEvent(_ context.Context, _ calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	p.got = ev
	return ev, nil
}

func TestCreateKeepsParticipationOnlyForImporters(t *testing.T) {
	doc, err := Parse([]byte(invitation))
	require.NoError(t, err)
	ev := doc.Events[0]
	cal := calendar.Calendar{ID: "cal-1"}

	t.Run("importer receives organizer and attendees", func(t *testing.T) {
		p := &importingProvider{MockProvider: mocks.NewMockProvider(t)}
		_, err := Create(context.Background(), p, cal, &ev)
		require.NoError(t, err)
		require.NotNil(t, p.got.Organizer)
		assert.Len(t, p.got.Attendees, 1)
	})

	t.Run("plain provider gets a private copy", func(t *testing.T) {
		p := mocks.NewMockProvider(t)
		p.EXPECT().CreateEvent(mock.Anything, cal, mock.MatchedBy(func(got *calendar.Event) bool {
			return got.Organizer == nil && got.Attendees == nil && got.UID == ev.UID && got.Summary == ev.Summary
		})).Return(&ev, nil)

		_, err := Create(context.Background(), p, cal, &ev)
		require.NoError(t, err)
		require.NotNil(t, ev.Organizer)
		assert.Len(t, ev.Attendees, 1)
	})
}
