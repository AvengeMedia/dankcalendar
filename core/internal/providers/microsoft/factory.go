package microsoft

import (
	"context"
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const (
	SecretKeyToken = "microsoft.token"
	SecretKeyApp   = "microsoft.app"
)

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountMicrosoft }

func (Factory) Build(ctx context.Context, account calendar.Account, secrets calendar.SecretStore) (calendar.Provider, error) {
	if secrets == nil {
		return nil, errors.New("microsoft provider requires a secret store")
	}
	return New(ctx, account, secrets)
}
