package accounts_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/internal/accounts"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/google"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func TestFlavor(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		settings map[string]any
		want     string
	}{
		{"google passes through", "google", nil, "google"},
		{"plain caldav", "caldav", map[string]any{"url": "https://dav.example.com"}, "caldav"},
		{"icloud preset", "caldav", map[string]any{"preset": "icloud"}, "icloud"},
		{"icloud url", "caldav", map[string]any{"url": "https://caldav.icloud.com"}, "icloud"},
		{"caldav without settings", "caldav", nil, "caldav"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, accounts.Flavor(tc.kind, tc.settings))
		})
	}
}

func TestProviderName(t *testing.T) {
	assert.Equal(t, "Google", accounts.ProviderName("google"))
	assert.Equal(t, "iCloud", accounts.ProviderName("icloud"))
	assert.Equal(t, "mystery", accounts.ProviderName("mystery"))
}

func TestAuthorized(t *testing.T) {
	ctx := context.Background()

	t.Run("local accounts need no credentials", func(t *testing.T) {
		secrets := mocks.NewMockSecretStore(t)
		acc := &ent.Account{ID: "loc", Kind: account.KindLocal}
		assert.True(t, accounts.Authorized(ctx, secrets, acc))
	})

	t.Run("google with stored token", func(t *testing.T) {
		secrets := mocks.NewMockSecretStore(t)
		secrets.EXPECT().Get(mock.Anything, "g", google.SecretKeyToken).Return([]byte("tok"), nil)
		acc := &ent.Account{ID: "g", Kind: account.KindGoogle}
		assert.True(t, accounts.Authorized(ctx, secrets, acc))
	})

	t.Run("google without token", func(t *testing.T) {
		secrets := mocks.NewMockSecretStore(t)
		secrets.EXPECT().Get(mock.Anything, "g", google.SecretKeyToken).Return(nil, errors.New("not found"))
		acc := &ent.Account{ID: "g", Kind: account.KindGoogle}
		assert.False(t, accounts.Authorized(ctx, secrets, acc))
	})
}

type AccountsSuite struct {
	suite.Suite
	ctx  context.Context
	repo *repo.Repo
}

func TestAccountsSuite(t *testing.T) {
	suite.Run(t, new(AccountsSuite))
}

func (s *AccountsSuite) SetupTest() {
	s.ctx = context.Background()
	client, err := repo.OpenMemory(s.ctx)
	s.Require().NoError(err)
	s.repo = repo.New(client)
	s.T().Cleanup(func() { _ = s.repo.Close() })
}

func (s *AccountsSuite) TestEnsureCreatesMissingAccount() {
	err := accounts.Ensure(s.ctx, s.repo, "personal", account.KindLocal, "Personal", map[string]any{"root": "/tmp"})
	s.Require().NoError(err)

	acc, err := s.repo.GetAccount(s.ctx, "personal")
	s.Require().NoError(err)
	s.Equal("Personal", acc.DisplayName)
}

func (s *AccountsSuite) TestEnsureIsIdempotent() {
	s.Require().NoError(accounts.Ensure(s.ctx, s.repo, "personal", account.KindLocal, "Personal", nil))
	s.Require().NoError(accounts.Ensure(s.ctx, s.repo, "personal", account.KindLocal, "Renamed", nil))

	acc, err := s.repo.GetAccount(s.ctx, "personal")
	s.Require().NoError(err)
	s.Equal("Personal", acc.DisplayName, "existing account should not be overwritten")

	all, err := s.repo.ListAccounts(s.ctx)
	s.Require().NoError(err)
	s.Len(all, 1)
}

func (s *AccountsSuite) TestDeleteRemovesAccountAndSecrets() {
	_, err := s.repo.CreateAccount(s.ctx, repo.CreateAccountInput{
		ID:          "g",
		Kind:        account.KindGoogle,
		DisplayName: "Google",
	})
	s.Require().NoError(err)

	secrets := repo.NewSecretStore(s.repo)
	s.Require().NoError(secrets.Set(s.ctx, "g", google.SecretKeyToken, []byte("tok")))

	s.Require().NoError(accounts.Delete(s.ctx, s.repo, secrets, "g"))

	_, err = s.repo.GetAccount(s.ctx, "g")
	s.True(repo.IsNotFound(err))

	_, err = secrets.Get(s.ctx, "g", google.SecretKeyToken)
	s.Error(err)
}
