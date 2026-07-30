package accounts

import (
	"testing"

	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

func TestResolveMicrosoftClientExplicitWins(t *testing.T) {
	t.Setenv("DANKCAL_MICROSOFT_CLIENT_ID", "env-id")

	got, err := ResolveMicrosoftClient(oauth.MicrosoftAppCredentials{ClientID: "my-id", Tenant: "contoso"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "my-id" || got.Tenant != "contoso" {
		t.Fatalf("got %+v, want explicit creds", got)
	}
}

func TestResolveMicrosoftClientEnvFallback(t *testing.T) {
	t.Setenv("DANKCAL_MICROSOFT_CLIENT_ID", "env-id")

	got, err := ResolveMicrosoftClient(oauth.MicrosoftAppCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "env-id" {
		t.Fatalf("got %+v, want env client ID", got)
	}
}

func TestResolveMicrosoftClientRejectsSecrets(t *testing.T) {
	t.Setenv("DANKCAL_MICROSOFT_CLIENT_ID", "")

	if _, err := ResolveMicrosoftClient(oauth.MicrosoftAppCredentials{ClientSecret: "secret"}); err == nil {
		t.Fatal("want error for client secret without ID")
	}
	if _, err := ResolveMicrosoftClient(oauth.MicrosoftAppCredentials{ClientID: "my-id", ClientSecret: "secret"}); err == nil {
		t.Fatal("want error for client secret with ID")
	}
}

func TestResolveMicrosoftClientBuiltinFallback(t *testing.T) {
	t.Setenv("DANKCAL_MICROSOFT_CLIENT_ID", "")

	got, err := ResolveMicrosoftClient(oauth.MicrosoftAppCredentials{Tenant: "contoso"})
	builtin, ok := oauth.BuiltinMicrosoftCredentials()
	switch {
	case !ok:
		if err == nil {
			t.Fatal("want error when no builtin client exists")
		}
	case err != nil:
		t.Fatal(err)
	case got.ClientID != builtin.ClientID || got.Tenant != "contoso":
		t.Fatalf("got %+v, want builtin ID with explicit tenant", got)
	}
}
