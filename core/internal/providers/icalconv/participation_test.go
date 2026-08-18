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

const recurringInviteICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//test//EN
BEGIN:VEVENT
UID:standup
DTSTAMP:20260101T000000Z
DTSTART:20260810T090000Z
DTEND:20260810T093000Z
SUMMARY:Standup
RRULE:FREQ=WEEKLY
ORGANIZER:mailto:boss@example.com
ATTENDEE;CN=Me;PARTSTAT=NEEDS-ACTION:mailto:me@example.com
ATTENDEE:mailto:peer@example.com
END:VEVENT
END:VCALENDAR
`

func participationDoc(t *testing.T, ics string) *ical.Calendar {
	t.Helper()
	doc, err := ical.NewDecoder(strings.NewReader(ics)).Decode()
	require.NoError(t, err)
	return doc
}

func attendeeStatuses(t *testing.T, comp *ical.Component) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, prop := range comp.Props.Values(ical.PropAttendee) {
		out[StripMailto(prop.Value)] = prop.Params.Get(ical.ParamParticipationStatus)
	}
	return out
}

func TestApplyParticipationSeriesPatchesMasterInPlace(t *testing.T) {
	doc := participationDoc(t, recurringInviteICS)
	ev := &cal.Event{UID: "standup", Start: time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)}

	rid, err := ApplyParticipation(doc, ev, "me@example.com", "accepted")
	require.NoError(t, err)
	assert.Empty(t, rid)
	require.Len(t, doc.Events(), 1)

	statuses := attendeeStatuses(t, doc.Events()[0].Component)
	assert.Equal(t, "ACCEPTED", statuses["me@example.com"])
	assert.Empty(t, statuses["peer@example.com"])
	assert.NotNil(t, doc.Events()[0].Props.Get(ical.PropRecurrenceRule))
}

func TestApplyParticipationOccurrenceCreatesException(t *testing.T) {
	doc := participationDoc(t, recurringInviteICS)
	occurrence := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	ev := &cal.Event{
		UID:           "standup/20260817T090000Z",
		RecurringID:   "standup",
		OriginalStart: occurrence,
		Start:         occurrence,
		End:           occurrence.Add(30 * time.Minute),
	}

	rid, err := ApplyParticipation(doc, ev, "me@example.com", "declined")
	require.NoError(t, err)
	assert.Equal(t, "20260817T090000Z", rid)
	require.Len(t, doc.Events(), 2)

	var buf bytes.Buffer
	require.NoError(t, ical.NewEncoder(&buf).Encode(doc))
	reparsed := participationDoc(t, buf.String())

	tz := NewTZResolver(reparsed, "")
	var master, exception cal.Event
	for _, comp := range reparsed.Events() {
		parsed, ok := EventFromComponent("cal", comp.Component, tz)
		require.True(t, ok)
		if parsed.RecurringID == "" {
			master = parsed
		} else {
			exception = parsed
		}
	}

	assert.Equal(t, "needs-action", master.Attendees[calIndex(master.Attendees, "me@example.com")].Status)
	assert.NotNil(t, master.Recurrence)

	assert.Equal(t, "standup/20260817T090000Z", exception.UID)
	assert.True(t, exception.OriginalStart.Equal(occurrence))
	assert.True(t, exception.Start.Equal(occurrence))
	assert.True(t, exception.End.Equal(occurrence.Add(30*time.Minute)))
	assert.Nil(t, exception.Recurrence)
	assert.Equal(t, "declined", exception.Attendees[calIndex(exception.Attendees, "me@example.com")].Status)
	assert.Equal(t, "Standup", exception.Summary)
}

func TestApplyParticipationOccurrencePatchesExistingException(t *testing.T) {
	doc := participationDoc(t, recurringInviteICS)
	occurrence := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	ev := &cal.Event{
		UID:           "standup/20260817T090000Z",
		RecurringID:   "standup",
		OriginalStart: occurrence,
		Start:         occurrence,
		End:           occurrence.Add(30 * time.Minute),
	}

	_, err := ApplyParticipation(doc, ev, "me@example.com", "tentative")
	require.NoError(t, err)
	rid, err := ApplyParticipation(doc, ev, "me@example.com", "accepted")
	require.NoError(t, err)
	assert.Equal(t, "20260817T090000Z", rid)
	require.Len(t, doc.Events(), 2)
}

func TestApplyParticipationRejectsNonAttendee(t *testing.T) {
	doc := participationDoc(t, recurringInviteICS)
	ev := &cal.Event{UID: "standup"}

	_, err := ApplyParticipation(doc, ev, "stranger@example.com", "accepted")
	assert.Error(t, err)
}

func TestApplyParticipationMissingEvent(t *testing.T) {
	doc := participationDoc(t, recurringInviteICS)
	ev := &cal.Event{UID: "other"}

	_, err := ApplyParticipation(doc, ev, "me@example.com", "accepted")
	assert.Error(t, err)
}

func calIndex(attendees []cal.Attendee, email string) int {
	for i := range attendees {
		if attendees[i].Email == email {
			return i
		}
	}
	return -1
}
