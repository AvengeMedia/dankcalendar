// Package eventconv maps between stored ent.Event rows and the domain
// calendar.Event model, keeping persistence and hydration identical across the
// sync engine, IPC handlers, and RSVP path.
package eventconv

import (
	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

// FromEnt hydrates the full domain event from a stored row, including the
// attendee, organizer, and recurrence data a remote update must not drop.
func FromEnt(e *ent.Event) calendar.Event {
	ev := calendar.Event{
		ID:            e.ID,
		UID:           e.UID,
		RemoteID:      e.RemoteID,
		Etag:          e.Etag,
		Summary:       e.Summary,
		Description:   e.Description,
		Location:      e.Location,
		URL:           e.URL,
		MeetingURL:    e.MeetingURL,
		Status:        calendar.EventStatus(e.Status),
		Start:         e.Start,
		End:           e.End,
		AllDay:        e.AllDay,
		StartTimeZone: e.StartTz,
		EndTimeZone:   e.EndTz,
		Recurrence:    calendar.RecurrenceFromMap(e.Recurrence),
		RecurringID:   e.RecurringID,
		Organizer:     calendar.OrganizerFromMap(e.Organizer),
		Attendees:     calendar.AttendeesFromMaps(e.Attendees),
		Reminders:     calendar.RemindersFromMaps(e.Reminders),
		Categories:    e.Categories,
		Transparency:  e.Transparency,
		Visibility:    e.Visibility,
		RawICS:        e.RawIcs,
	}
	if e.OriginalStart != nil {
		ev.OriginalStart = *e.OriginalStart
	}
	return ev
}

// UpsertInput builds the repo input that persists a domain event under a calendar.
func UpsertInput(calendarID string, ev *calendar.Event) repo.UpsertEventInput {
	var organizer map[string]any
	if ev.Organizer != nil {
		organizer = ev.Organizer.ToMap()
	}
	return repo.UpsertEventInput{
		CalendarID:    calendarID,
		UID:           ev.UID,
		RemoteID:      ev.RemoteID,
		Etag:          ev.Etag,
		Summary:       ev.Summary,
		Description:   ev.Description,
		Location:      ev.Location,
		URL:           ev.URL,
		MeetingURL:    ev.MeetingURL,
		Status:        EntStatus(ev.Status),
		Start:         ev.Start,
		End:           ev.End,
		AllDay:        ev.AllDay,
		StartTZ:       ev.StartTimeZone,
		EndTZ:         ev.EndTimeZone,
		Recurrence:    ev.Recurrence.ToMap(),
		RecurringID:   ev.RecurringID,
		OriginalStart: ev.OriginalStart,
		Organizer:     organizer,
		Attendees:     calendar.AttendeesToMaps(ev.Attendees),
		Reminders:     calendar.RemindersToMaps(ev.Reminders),
		Categories:    ev.Categories,
		Transparency:  ev.Transparency,
		Visibility:    ev.Visibility,
		RawICS:        ev.RawICS,
	}
}

func EntStatus(s calendar.EventStatus) event.Status {
	switch s {
	case calendar.EventTentative:
		return event.StatusTentative
	case calendar.EventCancelled:
		return event.StatusCancelled
	default:
		return event.StatusConfirmed
	}
}
