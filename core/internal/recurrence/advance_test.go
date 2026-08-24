package recurrence

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvanceDaily(t *testing.T) {
	next, rules, ok := Advance(Series{
		Start: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
		RRule: []string{"FREQ=DAILY"},
	})
	require.True(t, ok)
	assert.Equal(t, time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC), next)
	assert.Equal(t, []string{"FREQ=DAILY"}, rules)
}

func TestAdvanceDecrementsCount(t *testing.T) {
	next, rules, ok := Advance(Series{
		Start: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
		RRule: []string{"FREQ=WEEKLY;COUNT=3"},
	})
	require.True(t, ok)
	assert.Equal(t, time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC), next)
	assert.Equal(t, []string{"FREQ=WEEKLY;COUNT=2"}, rules)
}

func TestAdvanceExhaustedCount(t *testing.T) {
	_, _, ok := Advance(Series{
		Start: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
		RRule: []string{"FREQ=WEEKLY;COUNT=1"},
	})
	assert.False(t, ok)
}

func TestAdvancePastUntil(t *testing.T) {
	_, _, ok := Advance(Series{
		Start: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
		RRule: []string{"FREQ=DAILY;UNTIL=20260820T090000Z"},
	})
	assert.False(t, ok)
}

func TestAdvanceSkipsExDate(t *testing.T) {
	next, _, ok := Advance(Series{
		Start:  time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
		RRule:  []string{"FREQ=DAILY"},
		ExDate: []string{"20260821T090000Z"},
	})
	require.True(t, ok)
	assert.Equal(t, time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC), next)
}

func TestAdvanceNoRule(t *testing.T) {
	_, _, ok := Advance(Series{Start: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)})
	assert.False(t, ok)
}
