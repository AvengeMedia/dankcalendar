package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftAppCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	Tenant       string `json:"tenant,omitempty"`
}

func MicrosoftScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
		"offline_access",
		"https://graph.microsoft.com/Calendars.ReadWrite",
		"https://graph.microsoft.com/Tasks.ReadWrite",
		"https://graph.microsoft.com/User.Read",
	}
}

func MicrosoftConfig(creds MicrosoftAppCredentials, redirectURL string) *oauth2.Config {
	tenant := creds.Tenant
	if tenant == "" {
		tenant = "common"
	}
	return &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     microsoft.AzureADEndpoint(tenant),
		RedirectURL:  redirectURL,
		Scopes:       MicrosoftScopes(),
	}
}

type MicrosoftFlow struct {
	cfg      *oauth2.Config
	creds    MicrosoftAppCredentials
	callback Callback
}

func NewMicrosoftFlow(creds MicrosoftAppCredentials, callback Callback) (*MicrosoftFlow, error) {
	switch {
	case creds.ClientID == "":
		return nil, errors.New("microsoft client id is required")
	case callback == nil:
		return nil, errors.New("oauth callback is required")
	}

	return &MicrosoftFlow{
		cfg:      MicrosoftConfig(creds, callback.RedirectURL()),
		creds:    creds,
		callback: callback,
	}, nil
}

// StartMicrosoftLoopbackFlow runs the callback server in-process for use
// without a daemon. Azure only port-relaxes redirect URIs registered as
// http://localhost and matches the path exactly, so the callback lands on "/".
func StartMicrosoftLoopbackFlow(ctx context.Context, creds MicrosoftAppCredentials, bindAddr string) (*MicrosoftFlow, error) {
	if creds.ClientID == "" {
		return nil, errors.New("microsoft client id is required (see `dcal account setup microsoft` for instructions)")
	}

	loopback, err := StartLoopback(ctx, bindAddr)
	if err != nil {
		return nil, err
	}
	loopback.redirectHost = "localhost"
	loopback.redirectPath = "/"
	return NewMicrosoftFlow(creds, loopback)
}

func (m *MicrosoftFlow) AuthURL() string {
	return m.cfg.AuthCodeURL(m.callback.State(),
		oauth2.S256ChallengeOption(m.callback.Verifier()),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
}

func (m *MicrosoftFlow) Wait(ctx context.Context, timeout time.Duration) (*oauth2.Token, error) {
	defer m.callback.Close()

	res, err := m.callback.Wait(ctx, timeout)
	if err != nil {
		return nil, err
	}

	tok, err := m.cfg.Exchange(ctx, res.Code, oauth2.VerifierOption(m.callback.Verifier()))
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return tok, nil
}

func (m *MicrosoftFlow) Config() *oauth2.Config               { return m.cfg }
func (m *MicrosoftFlow) Credentials() MicrosoftAppCredentials { return m.creds }
func (m *MicrosoftFlow) State() string                        { return m.callback.State() }

func (m *MicrosoftFlow) Close() error { return m.callback.Close() }
