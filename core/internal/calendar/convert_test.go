package calendar_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

func TestAttendeeToMap(t *testing.T) {
	tests := []struct {
		name     string
		attendee calendar.Attendee
		want     map[string]any
	}{
		{
			"email only",
			calendar.Attendee{Email: "a@example.com"},
			map[string]any{"email": "a@example.com"},
		},
		{
			"all fields",
			calendar.Attendee{
				Email:       "a@example.com",
				DisplayName: "Alice",
				Role:        "REQ-PARTICIPANT",
				Status:      "accepted",
				Optional:    true,
				Organizer:   true,
			},
			map[string]any{
				"email":       "a@example.com",
				"displayName": "Alice",
				"role":        "REQ-PARTICIPANT",
				"status":      "accepted",
				"optional":    true,
				"organizer":   true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.attendee.ToMap())
		})
	}
}

func TestAttendeesToMaps(t *testing.T) {
	assert.Nil(t, calendar.AttendeesToMaps(nil))

	got := calendar.AttendeesToMaps([]calendar.Attendee{
		{Email: "a@example.com"},
		{Email: "b@example.com", Optional: true},
	})
	assert.Equal(t, []map[string]any{
		{"email": "a@example.com"},
		{"email": "b@example.com", "optional": true},
	}, got)
}

func TestRemindersToMaps(t *testing.T) {
	assert.Nil(t, calendar.RemindersToMaps(nil))

	got := calendar.RemindersToMaps([]calendar.Reminder{{Method: "popup", Minutes: 10}})
	assert.Equal(t, []map[string]any{{"method": "popup", "minutes": 10}}, got)
}

func TestRecurrenceToMap(t *testing.T) {
	tests := []struct {
		name string
		rec  *calendar.Recurrence
		want map[string]any
	}{
		{"nil receiver", nil, nil},
		{"empty", &calendar.Recurrence{}, nil},
		{
			"populated",
			&calendar.Recurrence{
				RRule:  []string{"FREQ=DAILY"},
				RDate:  []string{"20260101T000000Z"},
				ExDate: []string{"20260102T000000Z"},
			},
			map[string]any{
				"rrule":  []string{"FREQ=DAILY"},
				"rdate":  []string{"20260101T000000Z"},
				"exdate": []string{"20260102T000000Z"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.rec.ToMap())
		})
	}
}
