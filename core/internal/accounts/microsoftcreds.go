package accounts

import (
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

// ResolveMicrosoftClient rejects client secrets — desktop apps are public
// clients. Tenant may pin any credential source to one directory.
func ResolveMicrosoftClient(explicit oauth.MicrosoftAppCredentials) (oauth.MicrosoftAppCredentials, error) {
	if explicit.ClientSecret != "" {
		return oauth.MicrosoftAppCredentials{}, errors.New("microsoft desktop oauth clients are public clients and must not use a client secret")
	}
	if explicit.ClientID != "" {
		return explicit, nil
	}
	if clientID := config.New().MicrosoftClientID; clientID != "" {
		explicit.ClientID = clientID
		return explicit, nil
	}

	builtin, ok := oauth.BuiltinMicrosoftCredentials()
	if !ok {
		return oauth.MicrosoftAppCredentials{}, errors.New("this build has no built-in microsoft app registration; supply --client-id (see `dcal account setup microsoft`)")
	}
	builtin.Tenant = explicit.Tenant
	return builtin, nil
}
