package ipc

import (
	"testing"

	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

func TestParseGoogleStartParamsPassthrough(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "")

	got, err := parseGoogleStartParams(map[string]any{
		"displayName":  " Work ",
		"clientId":     " my-id ",
		"clientSecret": " my-secret ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Work" || got.ClientID != "my-id" || got.ClientSecret != "my-secret" {
		t.Fatalf("got %+v, want trimmed explicit params", got)
	}
}

func TestParseGoogleStartParamsPartialErrors(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "")

	if _, err := parseGoogleStartParams(map[string]any{"clientId": "my-id"}); err == nil {
		t.Fatal("want error for client ID without secret")
	}
}

func TestParseGoogleStartParamsResolvesDefaults(t *testing.T) {
	t.Setenv("DANKCAL_GOOGLE_CLIENT_ID", "")
	t.Setenv("DANKCAL_GOOGLE_CLIENT_SECRET", "")

	got, err := parseGoogleStartParams(map[string]any{})
	builtin, ok := oauth.BuiltinGoogleCredentials()
	switch {
	case !ok:
		if err == nil {
			t.Fatal("want error when no builtin client exists")
		}
	case err != nil:
		t.Fatal(err)
	case got.ClientID != builtin.ClientID || got.ClientSecret != builtin.ClientSecret:
		t.Fatalf("got %+v, want builtin creds", got)
	}
}
