package local

import (
	"context"
	"fmt"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountLocal }

func (Factory) Build(ctx context.Context, account calendar.Account, _ calendar.SecretStore) (calendar.Provider, error) {
	root, _ := account.Settings["root"].(string)
	if root == "" {
		return nil, fmt.Errorf("local account %q is missing the 'root' setting (path to ICS directory)", account.ID)
	}
	return New(account, root)
}
