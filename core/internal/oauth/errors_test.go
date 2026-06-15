package oauth

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/oauth2"
)

func TestFormatCallbackError(t *testing.T) {
	cases := []struct {
		name string
		code string
		desc string
		want string
	}{
		{"code only", "server_error", "", "oauth error: server_error"},
		{
			"code and description",
			"server_error",
			"AADSTS90013: Invalid input received from the user.",
			"oauth error: server_error: AADSTS90013: Invalid input received from the user.",
		},
		{
			"plus-encoded multiline description collapsed",
			"invalid_request",
			"AADSTS900144:+The+request+body+must+contain\r\nthe+parameter.",
			"oauth error: invalid_request: AADSTS900144: The request body must contain the parameter.",
		},
		{"description only", "", "something went wrong", "oauth error: something went wrong"},
		{"empty", "", "", "oauth error: authorization was denied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatCallbackError(tc.code, tc.desc); got != tc.want {
				t.Fatalf("FormatCallbackError(%q, %q) = %q, want %q", tc.code, tc.desc, got, tc.want)
			}
		})
	}
}

func TestIsInvalidGrant(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain", errors.New("network down"), false},
		{"invalid_grant", &oauth2.RetrieveError{ErrorCode: "invalid_grant"}, true},
		{"wrapped invalid_grant", fmt.Errorf("refresh: %w", &oauth2.RetrieveError{ErrorCode: "invalid_grant"}), true},
		{"other oauth error", &oauth2.RetrieveError{ErrorCode: "temporarily_unavailable"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsInvalidGrant(tc.err); got != tc.want {
				t.Fatalf("IsInvalidGrant(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
