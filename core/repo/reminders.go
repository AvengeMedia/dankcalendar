package repo

import (
	"context"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/reminderstate"
)

func (r *Repo) ListReminderStates(ctx context.Context, from, to time.Time) ([]*ent.ReminderState, error) {
	return r.client.ReminderState.Query().
		Where(
			reminderstate.OccurrenceStartGTE(from.UTC()),
			reminderstate.OccurrenceStartLTE(to.UTC()),
		).
		All(ctx)
}

// SetReminderFired upserts the state row for one (occurrence, offset) pair,
// stamping fired_at and clearing any pending snooze.
func (r *Repo) SetReminderFired(ctx context.Context, calendarID, uid string, occurrenceStart time.Time, minutes int, firedAt time.Time) error {
	existing, err := r.findReminderState(ctx, calendarID, uid, occurrenceStart, minutes)
	switch {
	case err == nil:
		return existing.Update().
			SetFiredAt(firedAt.UTC()).
			ClearSnoozedUntil().
			Exec(ctx)
	case !ent.IsNotFound(err):
		return err
	}

	return r.client.ReminderState.Create().
		SetCalendarID(calendarID).
		SetUID(uid).
		SetOccurrenceStart(occurrenceStart.UTC()).
		SetMinutes(minutes).
		SetFiredAt(firedAt.UTC()).
		Exec(ctx)
}

func (r *Repo) SnoozeReminder(ctx context.Context, calendarID, uid string, occurrenceStart time.Time, minutes int, until time.Time) error {
	existing, err := r.findReminderState(ctx, calendarID, uid, occurrenceStart, minutes)
	switch {
	case err == nil:
		return existing.Update().SetSnoozedUntil(until.UTC()).Exec(ctx)
	case !ent.IsNotFound(err):
		return err
	}

	return r.client.ReminderState.Create().
		SetCalendarID(calendarID).
		SetUID(uid).
		SetOccurrenceStart(occurrenceStart.UTC()).
		SetMinutes(minutes).
		SetSnoozedUntil(until.UTC()).
		Exec(ctx)
}

func (r *Repo) PruneReminderStates(ctx context.Context, before time.Time) (int, error) {
	return r.client.ReminderState.Delete().
		Where(reminderstate.OccurrenceStartLT(before.UTC())).
		Exec(ctx)
}

func (r *Repo) findReminderState(ctx context.Context, calendarID, uid string, occurrenceStart time.Time, minutes int) (*ent.ReminderState, error) {
	return r.client.ReminderState.Query().
		Where(
			reminderstate.CalendarIDEQ(calendarID),
			reminderstate.UIDEQ(uid),
			reminderstate.OccurrenceStartEQ(occurrenceStart.UTC()),
			reminderstate.MinutesEQ(minutes),
		).
		Only(ctx)
}
