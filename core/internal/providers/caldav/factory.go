package caldav

import (
	"context"
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const (
	SecretKeyPassword = "caldav.password"
)

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountCalDAV }

func (Factory) Build(ctx context.Context, account calendar.Account, secrets calendar.SecretStore) (calendar.Provider, error) {
	url, _ := account.Settings["url"].(string)
	username, _ := account.Settings["username"].(string)
	switch {
	case url == "":
		return nil, errors.New("caldav account requires 'url' setting")
	case username == "":
		return nil, errors.New("caldav account requires 'username' setting")
	}
	return New(ctx, account, secrets, url, username)
}
