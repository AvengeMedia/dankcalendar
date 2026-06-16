// Package oauthbase holds the OAuth scaffolding shared by token-based calendar
// providers: loading stored credentials into a self-persisting token source and
// tagging dead-credential failures for re-auth.
package oauthbase

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

// LoadTokenSource reads the account's stored app credentials and OAuth token and
// returns a token source that refreshes and re-persists the token on every use.
// C is the provider's app-credentials type; build turns it into the oauth2
// config used to refresh.
func LoadTokenSource[C any](ctx context.Context, secrets cal.SecretStore, account cal.Account, appKey, tokenKey string, build func(C) *oauth2.Config) (oauth2.TokenSource, error) {
	appBytes, err := secrets.Get(ctx, account.ID, appKey)
	if err != nil {
		return nil, fmt.Errorf("missing %s app credentials for account %q: %w", account.Kind, account.ID, err)
	}

	var creds C
	if err := json.Unmarshal(appBytes, &creds); err != nil {
		return nil, fmt.Errorf("decode %s app credentials: %w", account.Kind, err)
	}

	tokenBytes, err := secrets.Get(ctx, account.ID, tokenKey)
	if err != nil {
		return nil, fmt.Errorf("account %q is not authorized; run `dcal account %s login %s`", account.ID, account.Kind, account.ID)
	}

	tok, err := oauth.UnmarshalToken(tokenBytes)
	if err != nil {
		return nil, err
	}

	return &persistingTokenSource{
		base:      build(creds).TokenSource(ctx, tok),
		secrets:   secrets,
		accountID: account.ID,
		tokenKey:  tokenKey,
	}, nil
}

// ClassifyAuthErr wraps ErrReauthRequired around dead-credential failures so the
// sync engine flags the account: a refresh yielding invalid_grant, or a response
// the provider-specific predicate judges unauthorized.
func ClassifyAuthErr(err error, unauthorized func(error) bool) error {
	if err == nil {
		return nil
	}
	if oauth.IsInvalidGrant(err) || unauthorized(err) {
		return fmt.Errorf("%w: %v", cal.ErrReauthRequired, err)
	}
	return err
}

type persistingTokenSource struct {
	base      oauth2.TokenSource
	secrets   cal.SecretStore
	accountID string
	tokenKey  string
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if data, err := oauth.MarshalToken(tok); err == nil {
		_ = s.secrets.Set(context.Background(), s.accountID, s.tokenKey, data)
	}
	return tok, nil
}
