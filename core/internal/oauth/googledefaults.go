package oauth

import "encoding/base64"

// Shipped OAuth client for official builds (Google "Desktop app" client type,
// public per the installed-app model). Encoded — not encrypted — only so the
// values are not searchable plaintext in the repository; installed-app
// credentials cannot be kept confidential.
//
// These credentials identify the upstream DankCalendar application.
// Distribution packages built from this repository may ship them unchanged;
// forks, rebranded builds, and unrelated applications are not authorized to
// use them (see "OAuth client credentials" in the README) and must register
// their own client, overriding both vars via
// -ldflags "-X .../internal/oauth.builtinGoogleClientID=..." (plaintext), or
// blanking the encoded vars to require user-supplied credentials.
var (
	builtinGoogleClientID     = ""
	builtinGoogleClientSecret = ""
	encodedGoogleClientID     = "UlxZHFBRWl9YXVpMVUhVBQQNUBEdXQ8MDB0EH1QYCwUIGgdbDEVJSVYNH18TTw0VHhdPFQwACkIBFB0OHgoBGwxLChVACAwM"
	encodedGoogleClientSecret = "JCAufTQ5Qww+Hi0sGX9UORRcElI8PBYuFAMQFxxkLgcKCB4="
)

func BuiltinGoogleCredentials() (GoogleAppCredentials, bool) {
	switch {
	case builtinGoogleClientID != "" && builtinGoogleClientSecret != "":
		return GoogleAppCredentials{ClientID: builtinGoogleClientID, ClientSecret: builtinGoogleClientSecret}, true
	case builtinGoogleClientID != "" || builtinGoogleClientSecret != "":
		return GoogleAppCredentials{}, false
	}

	id := deobfuscate(encodedGoogleClientID)
	secret := deobfuscate(encodedGoogleClientSecret)
	if id == "" || secret == "" {
		return GoogleAppCredentials{}, false
	}
	return GoogleAppCredentials{ClientID: id, ClientSecret: secret}, true
}

func deobfuscate(encoded string) string {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	key := "com.danklinux.dankcalendar"
	for i := range raw {
		raw[i] ^= key[i%len(key)]
	}
	return string(raw)
}
