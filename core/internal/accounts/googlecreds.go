package accounts

import (
	"errors"

	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

// ResolveGoogleClient takes each credential source as a complete pair — an ID
// from one source is never combined with a secret from another.
func ResolveGoogleClient(explicit oauth.GoogleAppCredentials) (oauth.GoogleAppCredentials, error) {
	switch {
	case explicit.ClientID != "" && explicit.ClientSecret != "":
		return explicit, nil
	case explicit.ClientID != "" || explicit.ClientSecret != "":
		return oauth.GoogleAppCredentials{}, errors.New("custom google oauth clients need both a client ID and a client secret")
	}

	cfg := config.New()
	env := oauth.GoogleAppCredentials{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleSecret}
	switch {
	case env.ClientID != "" && env.ClientSecret != "":
		return env, nil
	case env.ClientID != "" || env.ClientSecret != "":
		return oauth.GoogleAppCredentials{}, errors.New("DANKCAL_GOOGLE_CLIENT_ID and DANKCAL_GOOGLE_CLIENT_SECRET must both be set")
	}

	builtin, ok := oauth.BuiltinGoogleCredentials()
	if !ok {
		return oauth.GoogleAppCredentials{}, errors.New("this build has no built-in google oauth client; supply --client-id and --client-secret (see `dcal account setup google`)")
	}
	return builtin, nil
}
