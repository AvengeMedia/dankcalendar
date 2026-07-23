package ipc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/recurrence"
)

func TestEventFromParamsRecurrence(t *testing.T) {
	base := calendar.Event{
		Summary: "standup",
		Start:   time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC),
		End:     time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC),
	}

	t.Run("sets rrule from list", func(t *testing.T) {
		ev, err := eventFromParams(base, map[string]any{"recurrence": []any{"FREQ=WEEKLY;BYDAY=MO,WE"}})
		require.NoError(t, err)
		require.NotNil(t, ev.Recurrence)
		assert.Equal(t, []string{"FREQ=WEEKLY;BYDAY=MO,WE"}, ev.Recurrence.RRule)
	})

	t.Run("empty list clears recurrence", func(t *testing.T) {
		withRule := base
		withRule.Recurrence = &calendar.Recurrence{RRule: []string{"FREQ=DAILY"}}
		ev, err := eventFromParams(withRule, map[string]any{"recurrence": []any{}})
		require.NoError(t, err)
		assert.Nil(t, ev.Recurrence)
	})

	t.Run("replacing rrule preserves rdate and exdate", func(t *testing.T) {
		withRule := base
		withRule.Recurrence = &calendar.Recurrence{
			RRule:  []string{"FREQ=DAILY"},
			RDate:  []string{"20260710T090000Z"},
			ExDate: []string{"20260708T090000Z"},
		}
		ev, err := eventFromParams(withRule, map[string]any{"recurrence": []any{"FREQ=WEEKLY"}})
		require.NoError(t, err)
		require.NotNil(t, ev.Recurrence)
		assert.Equal(t, []string{"FREQ=WEEKLY"}, ev.Recurrence.RRule)
		assert.Equal(t, []string{"20260710T090000Z"}, ev.Recurrence.RDate)
		assert.Equal(t, []string{"20260708T090000Z"}, ev.Recurrence.ExDate)
	})

	t.Run("absent key leaves recurrence untouched", func(t *testing.T) {
		withRule := base
		withRule.Recurrence = &calendar.Recurrence{RRule: []string{"FREQ=DAILY"}}
		ev, err := eventFromParams(withRule, map[string]any{"summary": "renamed"})
		require.NoError(t, err)
		require.NotNil(t, ev.Recurrence)
		assert.Equal(t, []string{"FREQ=DAILY"}, ev.Recurrence.RRule)
	})
}

func TestShiftSeriesTimes(t *testing.T) {
	masterStart := time.Date(2026, 7, 6, 14, 0, 0, 0, time.UTC)
	occStart := time.Date(2026, 7, 20, 14, 0, 0, 0, time.UTC)

	t.Run("unchanged occurrence times keep the master start", func(t *testing.T) {
		ev := shiftSeriesTimes(calendar.Event{
			Start: occStart,
			End:   occStart.Add(time.Hour),
		}, masterStart, occStart)
		assert.Equal(t, masterStart, ev.Start)
		assert.Equal(t, masterStart.Add(time.Hour), ev.End)
	})

	t.Run("moved occurrence shifts the series by the delta", func(t *testing.T) {
		ev := shiftSeriesTimes(calendar.Event{
			Start: occStart.Add(90 * time.Minute),
			End:   occStart.Add(90*time.Minute + 2*time.Hour),
		}, masterStart, occStart)
		assert.Equal(t, masterStart.Add(90*time.Minute), ev.Start)
		assert.Equal(t, masterStart.Add(90*time.Minute+2*time.Hour), ev.End)
	})
}

func TestExDateValue(t *testing.T) {
	occ := time.Date(2026, 7, 23, 19, 0, 0, 0, time.UTC)

	t.Run("timed uses UTC date-time form", func(t *testing.T) {
		assert.Equal(t, "20260723T190000Z", exDateValue(occ, false))
	})

	t.Run("zoned instant normalizes to UTC", func(t *testing.T) {
		ny, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "20260723T190000Z", exDateValue(occ.In(ny), false))
	})

	t.Run("all day uses date form", func(t *testing.T) {
		assert.Equal(t, "20260723", exDateValue(occ, true))
	})
}

// The exclusion written by deleteEventOccurrence must suppress exactly that
// occurrence when the series expands again.
func TestExDateValueExcludesOccurrence(t *testing.T) {
	series := recurrence.Series{
		Start: time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC),
		RRule: []string{"FREQ=DAILY"},
	}
	from := series.Start
	to := series.Start.AddDate(0, 0, 4)

	starts, err := recurrence.Expand(series, from, to)
	require.NoError(t, err)
	require.Len(t, starts, 5)

	series.ExDate = []string{exDateValue(starts[2], false)}
	starts, err = recurrence.Expand(series, from, to)
	require.NoError(t, err)
	require.Len(t, starts, 4)
	for _, s := range starts {
		assert.NotEqual(t, time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC), s.UTC())
	}
}

// Regression: domainEventFromEnt used to drop recurrence and original start,
// so any edit of a recurring master wiped its RRULE at the provider.
func TestDomainEventFromEntCarriesRecurrence(t *testing.T) {
	origStart := time.Date(2026, 7, 8, 9, 0, 0, 0, time.UTC)
	ev := domainEventFromEnt(&ent.Event{
		UID: "abc",
		Recurrence: map[string]any{
			"rrule":  []any{"FREQ=WEEKLY;BYDAY=MO"},
			"exdate": []any{"20260713T090000Z"},
		},
		OriginalStart: &origStart,
	})

	require.NotNil(t, ev.Recurrence)
	assert.Equal(t, []string{"FREQ=WEEKLY;BYDAY=MO"}, ev.Recurrence.RRule)
	assert.Equal(t, []string{"20260713T090000Z"}, ev.Recurrence.ExDate)
	assert.Equal(t, origStart, ev.OriginalStart)
}
