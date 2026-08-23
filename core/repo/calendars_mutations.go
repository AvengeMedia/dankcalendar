package repo

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/ent/task"
	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
)

type UpsertCalendarInput struct {
	ID                  string
	AccountID           string
	RemoteID            string
	Name                string
	Description         string
	Color               string
	TimeZone            string
	ReadOnly            bool
	Hidden              bool
	SyncToken           string
	SupportedComponents []string
}

func (r *Repo) UpsertCalendar(ctx context.Context, in UpsertCalendarInput) (*ent.Calendar, error) {
	existing, err := r.FindCalendarByRemoteID(ctx, in.AccountID, in.RemoteID)
	switch {
	case err == nil:
		// Hidden and sync_token are owned locally: hidden is a user
		// preference, the token is persisted via SetCalendarSyncToken.
		return r.calendarUpdate(existing.ID, in).Save(ctx)
	case !IsNotFound(err):
		return nil, err
	}

	id := in.ID
	if id == "" {
		id = newID()
	}
	create := r.client.Calendar.Create().
		SetID(id).
		SetAccountID(in.AccountID).
		SetRemoteID(in.RemoteID).
		SetName(in.Name).
		SetDescription(in.Description).
		SetColor(in.Color).
		SetTimeZone(in.TimeZone).
		SetReadOnly(in.ReadOnly).
		SetHidden(in.Hidden).
		SetSyncToken(in.SyncToken)
	if len(in.SupportedComponents) > 0 {
		create = create.SetSupportedComponents(in.SupportedComponents)
	}

	cal, err := create.Save(ctx)
	if err != nil && ent.IsConstraintError(err) {
		// A concurrent sync (e.g. the daemon and a manual `dcal sync`) inserted
		// the same (account, remote_id) between our find and create; fall back
		// to updating the row that now exists instead of failing the sync.
		if existing, ferr := r.FindCalendarByRemoteID(ctx, in.AccountID, in.RemoteID); ferr == nil {
			return r.calendarUpdate(existing.ID, in).Save(ctx)
		}
	}
	return cal, err
}

// calendarUpdate builds the update for provider-discovered fields. Components
// and color are only written when the provider reported them, so a discovery
// that omits them never clears a stored value.
func (r *Repo) calendarUpdate(id string, in UpsertCalendarInput) *ent.CalendarUpdateOne {
	upd := r.client.Calendar.UpdateOneID(id).
		SetName(in.Name).
		SetDescription(in.Description).
		SetTimeZone(in.TimeZone).
		SetReadOnly(in.ReadOnly)
	if in.Color != "" {
		upd = upd.SetColor(in.Color)
	}
	if len(in.SupportedComponents) > 0 {
		upd = upd.SetSupportedComponents(in.SupportedComponents)
	}
	return upd
}

// SetCalendarSyncToken is a no-op for disabled calendars so an in-flight sync
// cannot leave a stale cursor behind.
func (r *Repo) SetCalendarSyncToken(ctx context.Context, id, token string) error {
	_, err := r.client.Calendar.Update().
		Where(calendar.IDEQ(id), calendar.SyncDisabled(false)).
		SetSyncToken(token).
		Save(ctx)
	return err
}

func (r *Repo) SetCalendarHidden(ctx context.Context, id string, hidden bool) error {
	return r.client.Calendar.UpdateOneID(id).SetHidden(hidden).Exec(ctx)
}

// SetCalendarSyncDisabled excludes a calendar from provider sync. Disabling
// purges its local events and tasks and drops the sync token so a later
// re-enable starts from a fresh snapshot.
func (r *Repo) SetCalendarSyncDisabled(ctx context.Context, id string, disabled bool) error {
	if !disabled {
		return r.client.Calendar.UpdateOneID(id).
			SetSyncDisabled(false).
			ClearSyncToken().
			Exec(ctx)
	}
	return r.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := tx.Event.Delete().
			Where(event.HasCalendarWith(calendar.IDEQ(id))).
			Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Task.Delete().
			Where(task.HasCalendarWith(calendar.IDEQ(id))).
			Exec(ctx); err != nil {
			return err
		}
		return tx.Calendar.UpdateOneID(id).
			SetSyncDisabled(true).
			ClearSyncToken().
			Exec(ctx)
	})
}

func (r *Repo) SetCalendarColor(ctx context.Context, id, color string) error {
	return r.client.Calendar.UpdateOneID(id).SetColor(color).Exec(ctx)
}

func (r *Repo) SetCalendarNameOverride(ctx context.Context, id, name string) error {
	upd := r.client.Calendar.UpdateOneID(id)
	if name == "" {
		upd.ClearNameOverride()
	} else {
		upd.SetNameOverride(name)
	}
	return upd.Exec(ctx)
}

// SetCalendarReminders stores per-calendar reminder overrides. A nil or empty
// override clears the field, reverting the calendar to the global settings.
func (r *Repo) SetCalendarReminders(ctx context.Context, id string, o *settings.ReminderOverride) error {
	upd := r.client.Calendar.UpdateOneID(id)
	if o.IsEmpty() {
		upd.ClearReminderOverrides()
	} else {
		upd.SetReminderOverrides(o)
	}
	return upd.Exec(ctx)
}

func (r *Repo) DeleteCalendar(ctx context.Context, id string) error {
	return r.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := tx.Event.Delete().
			Where(event.HasCalendarWith(calendar.IDEQ(id))).
			Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Task.Delete().
			Where(task.HasCalendarWith(calendar.IDEQ(id))).
			Exec(ctx); err != nil {
			return err
		}
		_, err := tx.Calendar.Delete().
			Where(calendar.IDEQ(id)).
			Exec(ctx)
		return err
	})
}
