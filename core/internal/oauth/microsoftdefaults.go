package oauth

// Shipped Entra app registration for official builds (public client, PKCE only
// — client IDs are public identifiers, so no secret and no obfuscation).
// This identifies the upstream DankCalendar application: distribution packages
// built from this repository may ship it unchanged; forks and rebranded builds
// must register their own app and override via -ldflags -X, or blank this to
// require user-supplied credentials.
var builtinMicrosoftClientID = "d963e166-4fff-41b9-9e39-beee21f065bb"

func BuiltinMicrosoftCredentials() (MicrosoftAppCredentials, bool) {
	if builtinMicrosoftClientID == "" {
		return MicrosoftAppCredentials{}, false
	}
	return MicrosoftAppCredentials{ClientID: builtinMicrosoftClientID}, true
}
