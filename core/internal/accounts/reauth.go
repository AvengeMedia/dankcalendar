package accounts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/google"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/microsoft"
)

// GoogleAppCreds loads the OAuth client an account was added with, so a re-auth
// can reuse it instead of asking the user for the client ID and secret again.
func GoogleAppCreds(ctx context.Context, secrets calendar.SecretStore, accountID string) (oauth.GoogleAppCredentials, error) {
	appBytes, err := secrets.Get(ctx, accountID, google.SecretKeyApp)
	if err != nil {
		return oauth.GoogleAppCredentials{}, fmt.Errorf("no stored google credentials for %q; remove the account and add it again", accountID)
	}
	var creds oauth.GoogleAppCredentials
	if err := json.Unmarshal(appBytes, &creds); err != nil {
		return oauth.GoogleAppCredentials{}, fmt.Errorf("decode google credentials: %w", err)
	}
	return creds, nil
}

// MicrosoftAppCreds loads the app registration an account was added with.
func MicrosoftAppCreds(ctx context.Context, secrets calendar.SecretStore, accountID string) (oauth.MicrosoftAppCredentials, error) {
	appBytes, err := secrets.Get(ctx, accountID, microsoft.SecretKeyApp)
	if err != nil {
		return oauth.MicrosoftAppCredentials{}, fmt.Errorf("no stored microsoft credentials for %q; remove the account and add it again", accountID)
	}
	var creds oauth.MicrosoftAppCredentials
	if err := json.Unmarshal(appBytes, &creds); err != nil {
		return oauth.MicrosoftAppCredentials{}, fmt.Errorf("decode microsoft credentials: %w", err)
	}
	return creds, nil
}
