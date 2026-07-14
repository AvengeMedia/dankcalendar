package repo

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankgo/errdefs"
)

func (r *Repo) GetCalendar(ctx context.Context, id string) (*ent.Calendar, error) {
	c, err := r.client.Calendar.Query().Where(calendar.IDEQ(id)).WithAccount().Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "calendar not found")
	case err != nil:
		return nil, err
	}
	return c, nil
}

func (r *Repo) ListCalendars(ctx context.Context) ([]*ent.Calendar, error) {
	return r.client.Calendar.Query().
		WithAccount().
		Order(ent.Asc(calendar.FieldName)).
		All(ctx)
}

func (r *Repo) ListCalendarsForAccount(ctx context.Context, accountID string) ([]*ent.Calendar, error) {
	return r.client.Calendar.Query().
		Where(calendar.HasAccountWith(account.IDEQ(accountID))).
		WithAccount().
		Order(ent.Asc(calendar.FieldName)).
		All(ctx)
}

func (r *Repo) FindCalendarByRemoteID(ctx context.Context, accountID, remoteID string) (*ent.Calendar, error) {
	c, err := r.client.Calendar.Query().
		Where(
			calendar.HasAccountWith(account.IDEQ(accountID)),
			calendar.RemoteIDEQ(remoteID),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "calendar not found")
	case err != nil:
		return nil, err
	}
	return c, nil
}
