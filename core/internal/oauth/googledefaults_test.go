package oauth

import "testing"

func TestBuiltinGoogleCredentials(t *testing.T) {
	oldID, oldSecret := builtinGoogleClientID, builtinGoogleClientSecret
	oldEncodedID, oldEncodedSecret := encodedGoogleClientID, encodedGoogleClientSecret
	t.Cleanup(func() {
		builtinGoogleClientID, builtinGoogleClientSecret = oldID, oldSecret
		encodedGoogleClientID, encodedGoogleClientSecret = oldEncodedID, oldEncodedSecret
	})
	builtinGoogleClientID, builtinGoogleClientSecret = "", ""
	encodedGoogleClientID, encodedGoogleClientSecret = "", ""
	if _, ok := BuiltinGoogleCredentials(); ok {
		t.Fatal("empty build must require user-supplied credentials")
	}
	builtinGoogleClientID, builtinGoogleClientSecret = "test-client.apps.googleusercontent.com", "test-client-secret"
	creds, ok := BuiltinGoogleCredentials()
	if !ok || creds.ClientID != builtinGoogleClientID || creds.ClientSecret != builtinGoogleClientSecret {
		t.Fatal("build-time credentials not returned")
	}
	builtinGoogleClientSecret = ""
	if _, ok := BuiltinGoogleCredentials(); ok {
		t.Fatal("partial credentials must not be used")
	}
}
