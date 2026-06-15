package oauth

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/oauth2"
)

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
