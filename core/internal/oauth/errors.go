package oauth

import (
	"errors"

	"golang.org/x/oauth2"
)

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
