package repo

import (
	"context"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
)

type UpsertEventInput struct {
	ID            string
	CalendarID    string
	UID           string
	RemoteID      string
	Etag          string
	Summary       string
	Description   string
	Location      string
	URL           string
	Status        event.Status
	Start         time.Time
	End           time.Time
	AllDay        bool
	StartTZ       string
	EndTZ         string
	Recurrence    map[string]any
	RecurringID   string
	OriginalStart time.Time
	Organizer     map[string]any
	Attendees     []map[string]any
	Reminders     []map[string]any
	Categories    []string
	Transparency  string
	Visibility    string
	RawICS        string
}

func (r *Repo) UpsertEvent(ctx context.Context, in UpsertEventInput) (*ent.Event, error) {
	// Times are persisted as text; a single zone keeps range comparisons sane.
	in.Start = in.Start.UTC()
	in.End = in.End.UTC()
	in.OriginalStart = in.OriginalStart.UTC()

	existing, err := r.FindEventByUID(ctx, in.CalendarID, in.UID)
	switch {
	case err == nil:
		return r.applyEventUpdate(ctx, existing.ID, in)
	case !IsNotFound(err):
		return nil, err
	}

	id := in.ID
	if id == "" {
		id = newID()
	}
	q := r.client.Event.Create().
		SetID(id).
		SetCalendarID(in.CalendarID).
		SetUID(in.UID).
		SetRemoteID(in.RemoteID).
		SetEtag(in.Etag).
		SetSummary(in.Summary).
		SetDescription(in.Description).
		SetLocation(in.Location).
		SetURL(in.URL).
		SetStart(in.Start).
		SetEnd(in.End).
		SetAllDay(in.AllDay).
		SetStartTz(in.StartTZ).
		SetEndTz(in.EndTZ).
		SetRecurringID(in.RecurringID).
		SetTransparency(in.Transparency).
		SetVisibility(in.Visibility).
		SetRawIcs(in.RawICS)

	if in.Status != "" {
		q = q.SetStatus(in.Status)
	}
	if in.Recurrence != nil {
		q = q.SetRecurrence(in.Recurrence)
	}
	if !in.OriginalStart.IsZero() {
		q = q.SetOriginalStart(in.OriginalStart)
	}
	if in.Organizer != nil {
		q = q.SetOrganizer(in.Organizer)
	}
	if in.Attendees != nil {
		q = q.SetAttendees(in.Attendees)
	}
	if in.Reminders != nil {
		q = q.SetReminders(in.Reminders)
	}
	if in.Categories != nil {
		q = q.SetCategories(in.Categories)
	}
	return q.Save(ctx)
}

func (r *Repo) applyEventUpdate(ctx context.Context, id string, in UpsertEventInput) (*ent.Event, error) {
	q := r.client.Event.UpdateOneID(id).
		SetRemoteID(in.RemoteID).
		SetEtag(in.Etag).
		SetSummary(in.Summary).
		SetDescription(in.Description).
		SetLocation(in.Location).
		SetURL(in.URL).
		SetStart(in.Start).
		SetEnd(in.End).
		SetAllDay(in.AllDay).
		SetStartTz(in.StartTZ).
		SetEndTz(in.EndTZ).
		SetRecurringID(in.RecurringID).
		SetTransparency(in.Transparency).
		SetVisibility(in.Visibility).
		SetRawIcs(in.RawICS)

	if in.Status != "" {
		q = q.SetStatus(in.Status)
	}
	if in.Recurrence != nil {
		q = q.SetRecurrence(in.Recurrence)
	}
	if !in.OriginalStart.IsZero() {
		q = q.SetOriginalStart(in.OriginalStart)
	}
	if in.Organizer != nil {
		q = q.SetOrganizer(in.Organizer)
	}
	if in.Attendees != nil {
		q = q.SetAttendees(in.Attendees)
	}
	if in.Reminders != nil {
		q = q.SetReminders(in.Reminders)
	}
	if in.Categories != nil {
		q = q.SetCategories(in.Categories)
	}
	return q.Save(ctx)
}

func (r *Repo) DeleteEvent(ctx context.Context, id string) error {
	_, err := r.client.Event.Delete().Where(event.IDEQ(id)).Exec(ctx)
	return err
}

func (r *Repo) DeleteEventsNotInUIDs(ctx context.Context, calendarID string, uids []string) (int, error) {
	q := r.client.Event.Delete().
		Where(event.HasCalendarWith(calendar.IDEQ(calendarID)))
	if len(uids) > 0 {
		q = q.Where(event.UIDNotIn(uids...))
	}
	return q.Exec(ctx)
}

func (r *Repo) DeleteEventByUID(ctx context.Context, calendarID, uid string) error {
	_, err := r.client.Event.Delete().
		Where(
			event.HasCalendarWith(calendar.IDEQ(calendarID)),
			event.UIDEQ(uid),
		).
		Exec(ctx)
	return err
}
