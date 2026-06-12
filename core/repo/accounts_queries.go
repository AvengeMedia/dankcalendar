package repo

import (
	"context"
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/errdefs"
)

func (r *Repo) GetAccount(ctx context.Context, id string) (*ent.Account, error) {
	a, err := r.client.Account.Query().Where(account.IDEQ(id)).Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "account not found")
	case err != nil:
		return nil, err
	}
	return a, nil
}

func (r *Repo) ListAccounts(ctx context.Context) ([]*ent.Account, error) {
	return r.client.Account.Query().Order(ent.Asc(account.FieldCreatedAt)).All(ctx)
}

func (r *Repo) ListAccountsByKind(ctx context.Context, kind account.Kind) ([]*ent.Account, error) {
	return r.client.Account.Query().Where(account.KindEQ(kind)).All(ctx)
}

func (r *Repo) CountAccounts(ctx context.Context) (int, error) {
	return r.client.Account.Query().Count(ctx)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var custom *errdefs.CustomError
	switch {
	case errors.As(err, &custom):
		return custom.Type == errdefs.ErrTypeNotFound
	case ent.IsNotFound(err):
		return true
	}
	return false
}
