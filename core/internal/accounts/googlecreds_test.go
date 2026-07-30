package accounts

import (
	"testing"

	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

func TestResolveGoogleClientExplicitWins(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "env-id")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "env-secret")

	got, err := ResolveGoogleClient(oauth.GoogleAppCredentials{ClientID: "my-id", ClientSecret: "my-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "my-id" || got.ClientSecret != "my-secret" {
		t.Fatalf("got %+v, want explicit creds", got)
	}
}

func TestResolveGoogleClientEnvFallback(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "env-id")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "env-secret")

	got, err := ResolveGoogleClient(oauth.GoogleAppCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "env-id" || got.ClientSecret != "env-secret" {
		t.Fatalf("got %+v, want env creds", got)
	}
}

func TestResolveGoogleClientPartialErrors(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "")

	if _, err := ResolveGoogleClient(oauth.GoogleAppCredentials{ClientID: "my-id"}); err == nil {
		t.Fatal("want error for client ID without secret")
	}
	if _, err := ResolveGoogleClient(oauth.GoogleAppCredentials{ClientSecret: "my-secret"}); err == nil {
		t.Fatal("want error for client secret without ID")
	}
}

func TestResolveGoogleClientNeverMixesSources(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "env-secret")

	if _, err := ResolveGoogleClient(oauth.GoogleAppCredentials{ClientID: "my-id"}); err == nil {
		t.Fatal("want error instead of combining explicit ID with env secret")
	}
	if _, err := ResolveGoogleClient(oauth.GoogleAppCredentials{}); err == nil {
		t.Fatal("want error instead of combining partial env with builtin")
	}
}

func TestResolveGoogleClientBuiltinFallback(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "")

	got, err := ResolveGoogleClient(oauth.GoogleAppCredentials{})
	builtin, ok := oauth.BuiltinGoogleCredentials()
	switch {
	case !ok:
		if err == nil {
			t.Fatal("want error when no builtin client exists")
		}
	case err != nil:
		t.Fatal(err)
	case got != builtin:
		t.Fatalf("got %+v, want builtin creds", got)
	}
}
