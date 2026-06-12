package repo

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
)

type UpsertCalendarInput struct {
	ID          string
	AccountID   string
	RemoteID    string
	Name        string
	Description string
	Color       string
	TimeZone    string
	ReadOnly    bool
	Hidden      bool
	SyncToken   string
}

func (r *Repo) UpsertCalendar(ctx context.Context, in UpsertCalendarInput) (*ent.Calendar, error) {
	existing, err := r.FindCalendarByRemoteID(ctx, in.AccountID, in.RemoteID)
	switch {
	case err == nil:
		// Hidden and sync_token are owned locally: hidden is a user
		// preference, the token is persisted via SetCalendarSyncToken.
		return r.client.Calendar.UpdateOneID(existing.ID).
			SetName(in.Name).
			SetDescription(in.Description).
			SetColor(in.Color).
			SetTimeZone(in.TimeZone).
			SetReadOnly(in.ReadOnly).
			Save(ctx)
	case !IsNotFound(err):
		return nil, err
	}

	id := in.ID
	if id == "" {
		id = newID()
	}
	return r.client.Calendar.Create().
		SetID(id).
		SetAccountID(in.AccountID).
		SetRemoteID(in.RemoteID).
		SetName(in.Name).
		SetDescription(in.Description).
		SetColor(in.Color).
		SetTimeZone(in.TimeZone).
		SetReadOnly(in.ReadOnly).
		SetHidden(in.Hidden).
		SetSyncToken(in.SyncToken).
		Save(ctx)
}

func (r *Repo) SetCalendarSyncToken(ctx context.Context, id, token string) error {
	return r.client.Calendar.UpdateOneID(id).SetSyncToken(token).Exec(ctx)
}

func (r *Repo) SetCalendarHidden(ctx context.Context, id string, hidden bool) error {
	return r.client.Calendar.UpdateOneID(id).SetHidden(hidden).Exec(ctx)
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

func (r *Repo) DeleteCalendar(ctx context.Context, id string) error {
	return r.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := tx.Event.Delete().
			Where(event.HasCalendarWith(calendar.IDEQ(id))).
			Exec(ctx); err != nil {
			return err
		}
		_, err := tx.Calendar.Delete().
			Where(calendar.IDEQ(id)).
			Exec(ctx)
		return err
	})
}
