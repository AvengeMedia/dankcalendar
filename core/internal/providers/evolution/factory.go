package evolution

import (
	"context"

	"github.com/duxlabs-dev/dux-calendar/core/internal/calendar"
)

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountEvolution }

func (Factory) Build(ctx context.Context, account calendar.Account, _ calendar.SecretStore) (calendar.Provider, error) {
	return New(account)
}
