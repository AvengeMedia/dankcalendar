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

// roundTripTask encodes a task to ICS and parses it back, exercising both
// directions of the VTODO mapping.
func roundTripTask(t *testing.T, task *cal.Task, uid string) cal.Task {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, ical.NewEncoder(&buf).Encode(CalendarFromTask(task, uid)))

	doc, err := ical.NewDecoder(strings.NewReader(buf.String())).Decode()
	require.NoError(t, err)

	var todos []*ical.Component
	for _, child := range doc.Children {
		if child.Name == ical.CompToDo {
			todos = append(todos, child)
		}
	}
	require.Len(t, todos, 1)

	got, ok := TaskFromComponent("cal-1", todos[0], NewTZResolver(doc, ""))
	require.True(t, ok)
	return got
}

func TestRoundTripTaskCoreFields(t *testing.T) {
	src := &cal.Task{
		Summary:         "File taxes",
		Description:     "Before the deadline",
		Location:        "Home",
		Status:          cal.TaskInProcess,
		Priority:        5,
		PercentComplete: 40,
		Due:             time.Date(2026, 7, 1, 17, 0, 0, 0, time.UTC),
		Start:           time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		ParentUID:       "parent-1",
		Reminders:       []cal.Reminder{{Method: "popup", Minutes: 30}},
	}

	got := roundTripTask(t, src, "todo-1")
	assert.Equal(t, "todo-1", got.UID)
	assert.Equal(t, "cal-1", got.CalendarID)
	assert.Equal(t, src.Summary, got.Summary)
	assert.Equal(t, src.Description, got.Description)
	assert.Equal(t, src.Location, got.Location)
	assert.Equal(t, src.Status, got.Status)
	assert.Equal(t, src.Priority, got.Priority)
	assert.Equal(t, src.PercentComplete, got.PercentComplete)
	assert.Equal(t, src.Due, got.Due.UTC())
	assert.Equal(t, src.Start, got.Start.UTC())
	assert.Equal(t, src.ParentUID, got.ParentUID)
	require.Len(t, got.Reminders, 1)
	assert.Equal(t, 30, got.Reminders[0].Minutes)
}

func TestRoundTripTaskCompleted(t *testing.T) {
	src := &cal.Task{
		Summary:         "Pay rent",
		Status:          cal.TaskCompleted,
		PercentComplete: 100,
		Completed:       time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
	}

	got := roundTripTask(t, src, "todo-2")
	assert.Equal(t, cal.TaskCompleted, got.Status)
	assert.Equal(t, src.Completed, got.Completed.UTC())
}

func TestRoundTripTaskAllDayDue(t *testing.T) {
	src := &cal.Task{
		Summary: "Renew passport",
		AllDay:  true,
		Due:     time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	}

	got := roundTripTask(t, src, "todo-3")
	assert.True(t, got.AllDay)
	assert.Equal(t, src.Due, got.Due.UTC())
}

func TestTaskFromComponentNoUID(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//test//test//EN\r\n" +
		"BEGIN:VTODO\r\nSUMMARY:Orphan\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

	doc, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)
	require.Len(t, doc.Children, 1)

	_, ok := TaskFromComponent("cal-1", doc.Children[0], NewTZResolver(doc, ""))
	assert.False(t, ok, "a VTODO without a UID must be rejected")
}
