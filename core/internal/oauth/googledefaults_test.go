package oauth

import (
	"strings"
	"testing"
)

func TestBuiltinGoogleCredentials(t *testing.T) {
	creds, ok := BuiltinGoogleCredentials()
	if !ok {
		t.Fatal("no builtin credentials in default build")
	}
	if !strings.HasSuffix(creds.ClientID, ".apps.googleusercontent.com") {
		t.Fatalf("client ID has unexpected shape: %q", creds.ClientID)
	}
	if !strings.HasPrefix(creds.ClientSecret, "GOCS"+"PX-") {
		t.Fatal("client secret has unexpected shape")
	}
}
