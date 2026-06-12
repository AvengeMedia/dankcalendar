package google

import (
	"context"
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const (
	SecretKeyToken = "google.token"
	SecretKeyApp   = "google.app"
)

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountGoogle }

func (Factory) Build(ctx context.Context, account calendar.Account, secrets calendar.SecretStore) (calendar.Provider, error) {
	if secrets == nil {
		return nil, errors.New("google provider requires a secret store")
	}
	return New(ctx, account, secrets)
}
