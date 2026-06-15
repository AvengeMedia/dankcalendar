package repo

import (
	"context"
	"fmt"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/ent/secret"
)

type CreateAccountInput struct {
	ID          string
	Kind        account.Kind
	DisplayName string
	Settings    map[string]any
}

func (r *Repo) CreateAccount(ctx context.Context, in CreateAccountInput) (*ent.Account, error) {
	q := r.client.Account.Create().
		SetID(in.ID).
		SetKind(in.Kind).
		SetDisplayName(in.DisplayName)

	if len(in.Settings) > 0 {
		q = q.SetSettings(in.Settings)
	}
	return q.Save(ctx)
}

type UpdateAccountInput struct {
	DisplayName *string
	Settings    map[string]any
}

func (r *Repo) UpdateAccount(ctx context.Context, id string, in UpdateAccountInput) (*ent.Account, error) {
	q := r.client.Account.UpdateOneID(id)
	if in.DisplayName != nil {
		q = q.SetDisplayName(*in.DisplayName)
	}
	if in.Settings != nil {
		q = q.SetSettings(in.Settings)
	}
	return q.Save(ctx)
}

// SetAccountAuthState records whether an account needs re-authorization and the
// last auth error seen, so the CLI and GUI can surface a reconnect prompt.
func (r *Repo) SetAccountAuthState(ctx context.Context, id string, needsReauth bool, authError string) error {
	return r.client.Account.UpdateOneID(id).
		SetNeedsReauth(needsReauth).
		SetAuthError(authError).
		Exec(ctx)
}

// Children are deleted explicitly so pre-cascade databases stay deletable.
func (r *Repo) DeleteAccount(ctx context.Context, id string) error {
	return r.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := tx.Event.Delete().
			Where(event.HasCalendarWith(calendar.HasAccountWith(account.IDEQ(id)))).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete account events: %w", err)
		}
		if _, err := tx.Calendar.Delete().
			Where(calendar.HasAccountWith(account.IDEQ(id))).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete account calendars: %w", err)
		}
		if _, err := tx.Secret.Delete().
			Where(secret.HasAccountWith(account.IDEQ(id))).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete account secrets: %w", err)
		}
		if _, err := tx.Account.Delete().
			Where(account.IDEQ(id)).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete account: %w", err)
		}
		return nil
	})
}
