package icsimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const legacyCalendar = "BEGIN:VCALENDAR\r\nVERSION:1.0\r\nTZ:-07:00\r\nBEGIN:VEVENT\r\nDTSTART:20260910T120000\r\nDTEND:20260910T130000\r\nSUMMARY;ENCODING=QUOTED-PRINTABLE;CHARSET=UTF-8:Caf=C3=\r\n=A9 planning\r\nRRULE:D1 #3\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

func TestLegacyCalendar(t *testing.T) {
	doc, err := Parse([]byte(legacyCalendar))
	require.NoError(t, err)
	require.Len(t, doc.Events, 1)
	ev := doc.Events[0]
	assert.Equal(t, "Café planning", ev.Summary)
	assert.Equal(t, "2026-09-10T19:00:00Z", ev.Start.Format("2006-01-02T15:04:05Z"))
	assert.Equal(t, []string{"FREQ=DAILY;INTERVAL=1;COUNT=3"}, ev.Recurrence.RRule)
	again, err := Parse([]byte(legacyCalendar))
	require.NoError(t, err)
	assert.Equal(t, ev.UID, again.Events[0].UID)
	assert.NotEmpty(t, ev.UID)
}

func TestLegacyRules(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"D2 #0", "FREQ=DAILY;INTERVAL=2"},
		{"W2 MO WE FR #0", "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE,FR"},
		{"MP1 1- FR #10", "FREQ=MONTHLY;INTERVAL=1;COUNT=10;BYDAY=-1FR"},
		{"MD1 1 LD #0", "FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=1,-1"},
		{"YM1 6 #10", "FREQ=YEARLY;INTERVAL=1;COUNT=10;BYMONTH=6"},
		{"YD1 100 #0", "FREQ=YEARLY;INTERVAL=1;BYYEARDAY=100"},
	} {
		got, err := legacyRule(tc.input)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
	for _, raw := range []string{"", "D0", "W1 MO WE #3", "D1 #bad", "W1 BAD"} {
		_, err := legacyRule(raw)
		assert.Error(t, err, raw)
	}
}

func TestInvalidEventTimes(t *testing.T) {
	for _, input := range []string{
		strings.Replace(invitation, "20260910T140000", "bad", 1),
		strings.Replace(invitation, "20260910T150000", "20260909T150000", 1),
	} {
		_, err := Parse([]byte(input))
		assert.Error(t, err)
	}
}
