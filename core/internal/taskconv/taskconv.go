// Package taskconv maps between stored ent.Task rows and the domain
// calendar.Task model, keeping persistence and hydration identical across the
// sync engine and IPC handlers.
package taskconv

import (
	"github.com/AvengeMedia/dankcalendar/core/ent"
	enttask "github.com/AvengeMedia/dankcalendar/core/ent/task"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

// FromEnt hydrates the full domain task from a stored row.
func FromEnt(t *ent.Task) calendar.Task {
	out := calendar.Task{
		ID:              t.ID,
		UID:             t.UID,
		RemoteID:        t.RemoteID,
		Etag:            t.Etag,
		Summary:         t.Summary,
		Description:     t.Description,
		Location:        t.Location,
		Status:          calendar.TaskStatus(t.Status),
		Priority:        t.Priority,
		PercentComplete: t.PercentComplete,
		AllDay:          t.AllDay,
		DueTimeZone:     t.DueTz,
		StartTimeZone:   t.StartTz,
		ParentUID:       t.ParentUID,
		Recurrence:      calendar.RecurrenceFromMap(t.Recurrence),
		Reminders:       calendar.RemindersFromMaps(t.Reminders),
		Categories:      t.Categories,
		RawICS:          t.RawIcs,
		Created:         t.Created,
		Updated:         t.Updated,
	}
	if t.Due != nil {
		out.Due = *t.Due
	}
	if t.Start != nil {
		out.Start = *t.Start
	}
	if t.Completed != nil {
		out.Completed = *t.Completed
	}
	if t.Edges.Calendar != nil {
		out.CalendarID = t.Edges.Calendar.ID
	}
	return out
}

// UpsertInput builds the repo input that persists a domain task under a calendar.
func UpsertInput(calendarID string, t *calendar.Task) repo.UpsertTaskInput {
	return repo.UpsertTaskInput{
		CalendarID:      calendarID,
		UID:             t.UID,
		RemoteID:        t.RemoteID,
		Etag:            t.Etag,
		Summary:         t.Summary,
		Description:     t.Description,
		Location:        t.Location,
		Status:          EntStatus(t.Status),
		Priority:        t.Priority,
		PercentComplete: t.PercentComplete,
		Due:             t.Due,
		Start:           t.Start,
		Completed:       t.Completed,
		AllDay:          t.AllDay,
		DueTZ:           t.DueTimeZone,
		StartTZ:         t.StartTimeZone,
		ParentUID:       t.ParentUID,
		Recurrence:      t.Recurrence.ToMap(),
		Reminders:       calendar.RemindersToMaps(t.Reminders),
		Categories:      t.Categories,
		RawICS:          t.RawICS,
	}
}

func EntStatus(s calendar.TaskStatus) enttask.Status {
	switch s {
	case calendar.TaskInProcess:
		return enttask.StatusInProcess
	case calendar.TaskCompleted:
		return enttask.StatusCompleted
	case calendar.TaskCancelled:
		return enttask.StatusCancelled
	default:
		return enttask.StatusNeedsAction
	}
}
