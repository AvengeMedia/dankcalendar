package microsoft

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

func TestRruleFromGraphRecurrence(t *testing.T) {
	cases := []struct {
		name string
		in   *graphPatternedRecurrence
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{
			name: "daily",
			in:   &graphPatternedRecurrence{Pattern: graphRecurrencePattern{Type: "daily", Interval: 1}, Range: graphRecurrenceRange{Type: "noEnd"}},
			want: "FREQ=DAILY",
		},
		{
			name: "every 2 weeks on tue/thu",
			in:   &graphPatternedRecurrence{Pattern: graphRecurrencePattern{Type: "weekly", Interval: 2, DaysOfWeek: []string{"tuesday", "thursday"}}, Range: graphRecurrenceRange{Type: "noEnd"}},
			want: "FREQ=WEEKLY;INTERVAL=2;BYDAY=TU,TH",
		},
		{
			name: "monthly numbered",
			in:   &graphPatternedRecurrence{Pattern: graphRecurrencePattern{Type: "absoluteMonthly", Interval: 1, DayOfMonth: 15}, Range: graphRecurrenceRange{Type: "numbered", NumberOfOccurrences: 6}},
			want: "FREQ=MONTHLY;COUNT=6",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, rruleFromGraphRecurrence(tc.in))
		})
	}
}

func TestGraphRecurrenceFromTask(t *testing.T) {
	due := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)

	t.Run("no due date drops recurrence", func(t *testing.T) {
		task := &cal.Task{Recurrence: &cal.Recurrence{RRule: []string{"FREQ=DAILY"}}}
		require.Nil(t, graphRecurrenceFromTask(task))
	})

	t.Run("monthly anchors on the due day", func(t *testing.T) {
		task := &cal.Task{Due: due, Recurrence: &cal.Recurrence{RRule: []string{"FREQ=MONTHLY;INTERVAL=2"}}}
		got := graphRecurrenceFromTask(task)
		require.NotNil(t, got)
		require.Equal(t, "absoluteMonthly", got.Pattern.Type)
		require.Equal(t, 2, got.Pattern.Interval)
		require.Equal(t, 15, got.Pattern.DayOfMonth)
		require.Equal(t, "noEnd", got.Range.Type)
		require.Equal(t, "2026-03-15", got.Range.StartDate)
	})

	t.Run("weekly defaults to the due weekday", func(t *testing.T) {
		task := &cal.Task{Due: due, Recurrence: &cal.Recurrence{RRule: []string{"FREQ=WEEKLY"}}}
		got := graphRecurrenceFromTask(task)
		require.NotNil(t, got)
		require.Equal(t, "weekly", got.Pattern.Type)
		require.Equal(t, []string{"sunday"}, got.Pattern.DaysOfWeek)
	})

	t.Run("round-trips back to an equivalent rule", func(t *testing.T) {
		task := &cal.Task{Due: due, Recurrence: &cal.Recurrence{RRule: []string{"FREQ=WEEKLY;INTERVAL=2;BYDAY=TU,TH"}}}
		require.Equal(t, "FREQ=WEEKLY;INTERVAL=2;BYDAY=TU,TH", rruleFromGraphRecurrence(graphRecurrenceFromTask(task)))
	})
}
