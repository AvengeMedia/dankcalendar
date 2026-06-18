package recurrence_test

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/recurrence"
)

func TestExpandDailyRule(t *testing.T) {
	start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	series := recurrence.Series{Start: start, RRule: []string{"FREQ=DAILY;COUNT=3"}}

	got, err := recurrence.Expand(series, start, start.AddDate(0, 0, 30))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, start, got[0])
	assert.Equal(t, start.AddDate(0, 0, 2), got[2])
}

func TestExpandWindowIsInclusive(t *testing.T) {
	start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	series := recurrence.Series{Start: start, RRule: []string{"FREQ=DAILY"}}

	got, err := recurrence.Expand(series, start, start)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, start, got[0])
}

func TestExpandHonorsExDate(t *testing.T) {
	start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	series := recurrence.Series{
		Start:  start,
		RRule:  []string{"FREQ=DAILY;COUNT=3"},
		ExDate: []string{"20260106T090000Z"},
	}

	got, err := recurrence.Expand(series, start, start.AddDate(0, 0, 30))
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, start, got[0])
	assert.Equal(t, start.AddDate(0, 0, 2), got[1])
}

func TestExpandRDateOnly(t *testing.T) {
	start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	series := recurrence.Series{
		Start: start,
		RDate: []string{"20260110T090000Z,20260111T090000Z"},
	}

	got, err := recurrence.Expand(series, start, start.AddDate(0, 0, 30))
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC), got[0])
	assert.Equal(t, time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC), got[1])
}

func TestExpandFloatingExDateUsesSeriesTimeZone(t *testing.T) {
	start := time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC) // 09:00 in New York
	series := recurrence.Series{
		Start:    start,
		TimeZone: "America/New_York",
		RRule:    []string{"FREQ=DAILY;COUNT=2"},
		ExDate:   []string{"20260106T090000"},
	}

	got, err := recurrence.Expand(series, start, start.AddDate(0, 0, 30))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.True(t, got[0].Equal(start))
}

// A monthly "3rd Saturday" series anchored in Sydney must keep landing on
// Saturday 09:00 local on both sides of the DST switch. Expanding in UTC would
// drag occurrences onto Sunday and shift the hour, the symptoms in #13 and #14.
func TestExpandKeepsLocalWallClockAcrossDST(t *testing.T) {
	syd, err := time.LoadLocation("Australia/Sydney")
	require.NoError(t, err)

	start := time.Date(2025, 9, 20, 9, 0, 0, 0, syd) // 3rd Saturday, AEST
	series := recurrence.Series{
		Start:    start.UTC(),
		TimeZone: "Australia/Sydney",
		RRule:    []string{"FREQ=MONTHLY;BYDAY=3SA"},
	}

	got, err := recurrence.Expand(series, start.UTC(), start.AddDate(1, 0, 0).UTC())
	require.NoError(t, err)
	require.NotEmpty(t, got)

	for _, occ := range got {
		local := occ.In(syd)
		assert.Equal(t, time.Saturday, local.Weekday(), "occurrence %s", local)
		assert.Equal(t, 9, local.Hour(), "occurrence %s", local)
	}
}

// BYDAY rules anchored in a zone east of UTC must keep their weekdays and hour
// when expanded, regardless of which season they start in. These are the #14
// cases (and the reviewer's requested coverage): a UTC expansion would shift
// every occurrence to the next day.
func TestExpandByDayStableInZone(t *testing.T) {
	syd, err := time.LoadLocation("Australia/Sydney")
	require.NoError(t, err)

	cases := []struct {
		name     string
		start    time.Time
		rrule    string
		weekdays map[time.Weekday]bool
	}{
		{"weekdays", time.Date(2026, 1, 5, 9, 0, 0, 0, syd), "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
			map[time.Weekday]bool{time.Monday: true, time.Tuesday: true, time.Wednesday: true, time.Thursday: true, time.Friday: true}},
		{"mon wed fri", time.Date(2026, 1, 5, 9, 0, 0, 0, syd), "FREQ=WEEKLY;BYDAY=MO,WE,FR",
			map[time.Weekday]bool{time.Monday: true, time.Wednesday: true, time.Friday: true}},
		{"first friday", time.Date(2026, 1, 2, 9, 0, 0, 0, syd), "FREQ=MONTHLY;BYDAY=1FR",
			map[time.Weekday]bool{time.Friday: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := recurrence.Series{Start: tc.start.UTC(), TimeZone: "Australia/Sydney", RRule: []string{tc.rrule}}
			got, err := recurrence.Expand(s, tc.start.UTC(), tc.start.AddDate(1, 0, 0).UTC())
			require.NoError(t, err)
			require.NotEmpty(t, got)
			for _, occ := range got {
				local := occ.In(syd)
				assert.True(t, tc.weekdays[local.Weekday()], "unexpected weekday %s at %s", local.Weekday(), local)
				assert.Equal(t, 9, local.Hour(), "occurrence %s", local)
			}
		})
	}
}

// The #13 hour shift, verified in both directions across the DST boundary: a
// weekly 09:00 series started in summer must still read 09:00 after the autumn
// switch, and one started in winter must still read 09:00 after spring.
func TestExpandWeeklyHourStableBothDirections(t *testing.T) {
	syd, err := time.LoadLocation("Australia/Sydney")
	require.NoError(t, err)

	for _, start := range []time.Time{
		time.Date(2026, 1, 6, 9, 0, 0, 0, syd), // summer (AEDT)
		time.Date(2026, 7, 7, 9, 0, 0, 0, syd), // winter (AEST)
	} {
		s := recurrence.Series{Start: start.UTC(), TimeZone: "Australia/Sydney", RRule: []string{"FREQ=WEEKLY"}}
		got, err := recurrence.Expand(s, start.UTC(), start.AddDate(1, 0, 0).UTC())
		require.NoError(t, err)
		require.NotEmpty(t, got)
		for _, occ := range got {
			assert.Equal(t, 9, occ.In(syd).Hour(), "occurrence %s", occ.In(syd))
		}
	}
}

func TestExpandErrors(t *testing.T) {
	start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		series recurrence.Series
	}{
		{"empty series", recurrence.Series{Start: start}},
		{"only invalid rrule", recurrence.Series{Start: start, RRule: []string{"garbage"}}},
		{"only invalid rdate", recurrence.Series{Start: start, RDate: []string{"not-a-date"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := recurrence.Expand(tc.series, start, start.AddDate(0, 0, 30))
			require.ErrorIs(t, err, recurrence.ErrNoRule)
		})
	}
}

func TestParseDateTime(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	tests := []struct {
		name    string
		raw     string
		want    time.Time
		wantErr bool
	}{
		{"utc datetime", "20260507T140000Z", time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC), false},
		{"date only", "20260507", time.Date(2026, 5, 7, 0, 0, 0, 0, loc), false},
		{"floating datetime", "20260507T140000", time.Date(2026, 5, 7, 14, 0, 0, 0, loc), false},
		{"invalid", "bogus", time.Time{}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := recurrence.ParseDateTime(tc.raw, loc)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, got.Equal(tc.want), "got %v, want %v", got, tc.want)
		})
	}
}
