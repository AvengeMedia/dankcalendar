package repo

import (
	"context"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankcalendar/core/ent/task"
)

type UpsertTaskInput struct {
	ID              string
	CalendarID      string
	UID             string
	RemoteID        string
	Etag            string
	Summary         string
	Description     string
	Location        string
	Status          task.Status
	Priority        int
	PercentComplete int
	Due             time.Time
	Start           time.Time
	Completed       time.Time
	AllDay          bool
	DueTZ           string
	StartTZ         string
	ParentUID       string
	Recurrence      map[string]any
	Reminders       []map[string]any
	Categories      []string
	RawICS          string
}

func (r *Repo) UpsertTask(ctx context.Context, in UpsertTaskInput) (*ent.Task, error) {
	existing, err := r.FindTaskByUID(ctx, in.CalendarID, in.UID)
	switch {
	case err == nil:
		return r.applyTaskUpdate(ctx, existing.ID, in)
	case !IsNotFound(err):
		return nil, err
	}

	id := in.ID
	if id == "" {
		id = newID()
	}
	q := r.client.Task.Create().
		SetID(id).
		SetCalendarID(in.CalendarID).
		SetUID(in.UID).
		SetRemoteID(in.RemoteID).
		SetEtag(in.Etag).
		SetSummary(in.Summary).
		SetDescription(in.Description).
		SetLocation(in.Location).
		SetPriority(in.Priority).
		SetPercentComplete(in.PercentComplete).
		SetAllDay(in.AllDay).
		SetDueTz(in.DueTZ).
		SetStartTz(in.StartTZ).
		SetParentUID(in.ParentUID).
		SetRawIcs(in.RawICS)

	applyTaskOptionals(q.Mutation(), in)

	t, err := q.Save(ctx)
	if err != nil && ent.IsConstraintError(err) {
		// A concurrent sync inserted the same (calendar, uid) between our find
		// and create; fall back to updating the row that now exists.
		if existing, ferr := r.FindTaskByUID(ctx, in.CalendarID, in.UID); ferr == nil {
			return r.applyTaskUpdate(ctx, existing.ID, in)
		}
	}
	return t, err
}

func (r *Repo) applyTaskUpdate(ctx context.Context, id string, in UpsertTaskInput) (*ent.Task, error) {
	q := r.client.Task.UpdateOneID(id).
		SetRemoteID(in.RemoteID).
		SetEtag(in.Etag).
		SetSummary(in.Summary).
		SetDescription(in.Description).
		SetLocation(in.Location).
		SetPriority(in.Priority).
		SetPercentComplete(in.PercentComplete).
		SetAllDay(in.AllDay).
		SetDueTz(in.DueTZ).
		SetStartTz(in.StartTZ).
		SetParentUID(in.ParentUID).
		SetRawIcs(in.RawICS)

	applyTaskOptionals(q.Mutation(), in)
	return q.Save(ctx)
}

// applyTaskOptionals sets fields whose zero value carries meaning: status and
// the three nullable timestamps are set-or-cleared explicitly, while a nil JSON
// column leaves the stored value untouched.
func applyTaskOptionals(m *ent.TaskMutation, in UpsertTaskInput) {
	if in.Status != "" {
		m.SetStatus(in.Status)
	}
	if in.Due.IsZero() {
		m.ClearDue()
	} else {
		m.SetDue(in.Due.UTC())
	}
	if in.Start.IsZero() {
		m.ClearStart()
	} else {
		m.SetStart(in.Start.UTC())
	}
	if in.Completed.IsZero() {
		m.ClearCompleted()
	} else {
		m.SetCompleted(in.Completed.UTC())
	}
	if in.Recurrence != nil {
		m.SetRecurrence(in.Recurrence)
	}
	if in.Reminders != nil {
		m.SetReminders(in.Reminders)
	}
	if in.Categories != nil {
		m.SetCategories(in.Categories)
	}
}

func (r *Repo) DeleteTask(ctx context.Context, id string) error {
	_, err := r.client.Task.Delete().Where(task.IDEQ(id)).Exec(ctx)
	return err
}

func (r *Repo) DeleteTasksNotInUIDs(ctx context.Context, calendarID string, uids []string) (int, error) {
	q := r.client.Task.Delete().
		Where(task.HasCalendarWith(calendar.IDEQ(calendarID)))
	if len(uids) > 0 {
		q = q.Where(task.UIDNotIn(uids...))
	}
	return q.Exec(ctx)
}

func (r *Repo) DeleteTaskByUID(ctx context.Context, calendarID, uid string) error {
	_, err := r.client.Task.Delete().
		Where(
			task.HasCalendarWith(calendar.IDEQ(calendarID)),
			task.UIDEQ(uid),
		).
		Exec(ctx)
	return err
}
