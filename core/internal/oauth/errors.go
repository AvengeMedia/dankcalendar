package oauth

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
)

// FormatCallbackError builds a human-facing message from the error and
// error_description query parameters returned on an OAuth redirect. Providers
// (Microsoft especially) pack the actionable detail — and an AADSTS code — into
// error_description, so surfacing just the bare error code (e.g. "server_error")
// hides the real reason. The description is URL-decoded with literal "+" for
// spaces and may carry newlines, so collapse it to a single line.
func FormatCallbackError(errCode, desc string) string {
	errCode = strings.TrimSpace(errCode)
	desc = strings.Join(strings.Fields(strings.ReplaceAll(desc, "+", " ")), " ")
	switch {
	case errCode != "" && desc != "":
		return fmt.Sprintf("oauth error: %s: %s", errCode, desc)
	case errCode != "":
		return fmt.Sprintf("oauth error: %s", errCode)
	case desc != "":
		return fmt.Sprintf("oauth error: %s", desc)
	default:
		return "oauth error: authorization was denied"
	}
}

// IsInvalidGrant reports whether err is an OAuth token refresh that failed
// because the refresh token itself is dead (revoked, expired, or invalidated
// by a password change). Transient failures surface as other error types, so
// this distinguishes "needs re-auth" from "try again later".
func IsInvalidGrant(err error) bool {
	var re *oauth2.RetrieveError
	if !errors.As(err, &re) {
		return false
	}
	switch re.ErrorCode {
	case "invalid_grant", "invalid_token", "unauthorized_client":
		return true
	}
	return false
}
