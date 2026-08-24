package ipc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

func TestAdvanceRecurringTask(t *testing.T) {
	due := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	t.Run("moves due to next occurrence", func(t *testing.T) {
		task := calendar.Task{
			Due:        due,
			AllDay:     true,
			Status:     calendar.TaskCompleted,
			Recurrence: &calendar.Recurrence{RRule: []string{"FREQ=WEEKLY"}},
		}
		require.True(t, advanceRecurringTask(&task))
		assert.Equal(t, due.AddDate(0, 0, 7), task.Due)
		assert.Equal(t, calendar.TaskNeedsAction, task.Status)
		assert.Zero(t, task.PercentComplete)
		assert.True(t, task.Completed.IsZero())
	})

	t.Run("shifts start and due together", func(t *testing.T) {
		task := calendar.Task{
			Start:      due.Add(-24 * time.Hour),
			Due:        due,
			Recurrence: &calendar.Recurrence{RRule: []string{"FREQ=DAILY"}},
		}
		require.True(t, advanceRecurringTask(&task))
		assert.Equal(t, due, task.Start)
		assert.Equal(t, due.Add(24*time.Hour), task.Due)
	})

	t.Run("exhausted series completes", func(t *testing.T) {
		task := calendar.Task{
			Due:        due,
			Recurrence: &calendar.Recurrence{RRule: []string{"FREQ=DAILY;COUNT=1"}},
		}
		assert.False(t, advanceRecurringTask(&task))
	})

	t.Run("non-recurring completes", func(t *testing.T) {
		task := calendar.Task{Due: due}
		assert.False(t, advanceRecurringTask(&task))
	})

	t.Run("recurring without dates completes", func(t *testing.T) {
		task := calendar.Task{Recurrence: &calendar.Recurrence{RRule: []string{"FREQ=DAILY"}}}
		assert.False(t, advanceRecurringTask(&task))
	})
}
