package evolution

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountEvolution }

func (Factory) Build(ctx context.Context, account calendar.Account, _ calendar.SecretStore) (calendar.Provider, error) {
	return New(account)
}
