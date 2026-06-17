package ical

import (
	"context"
	"fmt"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const SecretKeyPassword = "ical.password"

type Factory struct{}

func (Factory) Kind() calendar.AccountKind { return calendar.AccountICal }

func (Factory) Build(ctx context.Context, account calendar.Account, secrets calendar.SecretStore) (calendar.Provider, error) {
	rawURL, _ := account.Settings["url"].(string)
	feedURL, err := normalizeURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("ical account: %w", err)
	}

	username, _ := account.Settings["username"].(string)
	var password []byte
	if username != "" {
		password, err = secrets.Get(ctx, account.ID, SecretKeyPassword)
		if err != nil {
			return nil, fmt.Errorf("ical password not stored for %q: %w", account.ID, err)
		}
	}

	name, _ := account.Settings["name"].(string)
	return New(account, feedURL, username, password, name), nil
}
