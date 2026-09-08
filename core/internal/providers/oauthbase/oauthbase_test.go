package oauthbase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
)

type countingSource struct {
	calls int
	tok   oauth2.Token
}

func (c *countingSource) Token() (*oauth2.Token, error) {
	c.calls++
	t := c.tok
	return &t, nil
}

func TestValidTokenNeverWrites(t *testing.T) {
	stored, err := json.Marshal(oauth2.Token{AccessToken: "a", Expiry: time.Now().Add(time.Hour)})
	require.NoError(t, err)

	store := mocks.NewMockSecretStore(t)
	store.EXPECT().Get(mock.Anything, "acct", "app").Return([]byte(`{}`), nil)
	store.EXPECT().Get(mock.Anything, "acct", "token").Return(stored, nil)

	src, err := LoadTokenSource(context.Background(), store, cal.Account{ID: "acct"}, "app", "token", func(struct{}) *oauth2.Config {
		return &oauth2.Config{}
	})
	require.NoError(t, err)

	for range 5 {
		_, err := src.Token()
		require.NoError(t, err)
	}
}

func TestRefreshWritesOnce(t *testing.T) {
	var written []byte
	store := mocks.NewMockSecretStore(t)
	store.EXPECT().Set(mock.Anything, "acct", "token", mock.Anything).
		RunAndReturn(func(_ context.Context, _, _ string, data []byte) error {
			written = data
			return nil
		}).Once()

	refresher := &countingSource{tok: oauth2.Token{AccessToken: "new", Expiry: time.Now().Add(time.Hour)}}
	src := oauth2.ReuseTokenSource(nil, &persistingTokenSource{base: refresher, secrets: store, accountID: "acct", tokenKey: "token"})

	for range 5 {
		_, err := src.Token()
		require.NoError(t, err)
	}
	require.Equal(t, 1, refresher.calls)

	var tok oauth2.Token
	require.NoError(t, json.Unmarshal(written, &tok))
	require.Equal(t, "new", tok.AccessToken)
}
