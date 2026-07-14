package repo

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/ent/secret"
	"github.com/AvengeMedia/dankgo/errdefs"
)

func (r *Repo) GetSecret(ctx context.Context, accountID, key string) ([]byte, error) {
	s, err := r.client.Secret.Query().
		Where(
			secret.HasAccountWith(account.IDEQ(accountID)),
			secret.KeyEQ(key),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "secret not found")
	case err != nil:
		return nil, err
	}
	return s.Value, nil
}

func (r *Repo) SetSecret(ctx context.Context, accountID, key string, value []byte) error {
	existing, err := r.client.Secret.Query().
		Where(
			secret.HasAccountWith(account.IDEQ(accountID)),
			secret.KeyEQ(key),
		).
		Only(ctx)

	switch {
	case ent.IsNotFound(err):
		_, err = r.client.Secret.Create().
			SetID(newID()).
			SetAccountID(accountID).
			SetKey(key).
			SetValue(value).
			Save(ctx)
		return err
	case err != nil:
		return err
	}
	return r.client.Secret.UpdateOneID(existing.ID).SetValue(value).Exec(ctx)
}

func (r *Repo) DeleteSecret(ctx context.Context, accountID, key string) error {
	_, err := r.client.Secret.Delete().
		Where(
			secret.HasAccountWith(account.IDEQ(accountID)),
			secret.KeyEQ(key),
		).
		Exec(ctx)
	return err
}

type SecretStore struct {
	repo *Repo
}

func NewSecretStore(repo *Repo) *SecretStore {
	return &SecretStore{repo: repo}
}

func (s *SecretStore) Get(ctx context.Context, accountID, key string) ([]byte, error) {
	return s.repo.GetSecret(ctx, accountID, key)
}

func (s *SecretStore) Set(ctx context.Context, accountID, key string, value []byte) error {
	return s.repo.SetSecret(ctx, accountID, key, value)
}

func (s *SecretStore) Delete(ctx context.Context, accountID, key string) error {
	return s.repo.DeleteSecret(ctx, accountID, key)
}
