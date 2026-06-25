package repo

import (
	"context"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/invitationstate"
)

// ListInvitationStates returns the notify-once records for invitations whose
// event starts within [from, to].
func (r *Repo) ListInvitationStates(ctx context.Context, from, to time.Time) ([]*ent.InvitationState, error) {
	return r.client.InvitationState.Query().
		Where(
			invitationstate.EventStartGTE(from.UTC()),
			invitationstate.EventStartLTE(to.UTC()),
		).
		All(ctx)
}

// SetInvitationNotified records that the user was prompted about an invitation,
// upserting the (calendar, uid) row.
func (r *Repo) SetInvitationNotified(ctx context.Context, calendarID, uid string, eventStart, notifiedAt time.Time) error {
	existing, err := r.client.InvitationState.Query().
		Where(
			invitationstate.CalendarIDEQ(calendarID),
			invitationstate.UIDEQ(uid),
		).
		Only(ctx)
	switch {
	case err == nil:
		return existing.Update().
			SetEventStart(eventStart.UTC()).
			SetNotifiedAt(notifiedAt.UTC()).
			Exec(ctx)
	case !ent.IsNotFound(err):
		return err
	}

	return r.client.InvitationState.Create().
		SetCalendarID(calendarID).
		SetUID(uid).
		SetEventStart(eventStart.UTC()).
		SetNotifiedAt(notifiedAt.UTC()).
		Exec(ctx)
}

func (r *Repo) PruneInvitationStates(ctx context.Context, before time.Time) (int, error) {
	return r.client.InvitationState.Delete().
		Where(invitationstate.EventStartLT(before.UTC())).
		Exec(ctx)
}
