package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gcalendar "google.golang.org/api/calendar/v3"
	gtasks "google.golang.org/api/tasks/v1"
)

type GoogleAppCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func GoogleScopes() []string {
	return []string{
		gcalendar.CalendarEventsScope,
		gcalendar.CalendarCalendarlistReadonlyScope,
		gtasks.TasksScope,
		"https://www.googleapis.com/auth/userinfo.email",
		"openid",
	}
}

func GoogleConfig(creds GoogleAppCredentials, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       GoogleScopes(),
	}
}

type GoogleFlow struct {
	cfg      *oauth2.Config
	creds    GoogleAppCredentials
	callback Callback
}

func NewGoogleFlow(creds GoogleAppCredentials, callback Callback) (*GoogleFlow, error) {
	switch {
	case creds.ClientID == "" || creds.ClientSecret == "":
		return nil, errors.New("google app credentials are required")
	case callback == nil:
		return nil, errors.New("oauth callback is required")
	}

	return &GoogleFlow{
		cfg:      GoogleConfig(creds, callback.RedirectURL()),
		creds:    creds,
		callback: callback,
	}, nil
}

func StartGoogleLoopbackFlow(ctx context.Context, creds GoogleAppCredentials, bindAddr string) (*GoogleFlow, error) {
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return nil, errors.New("google app credentials are required (see `dcal account setup google` for instructions)")
	}

	loopback, err := StartLoopback(ctx, bindAddr)
	if err != nil {
		return nil, err
	}
	return NewGoogleFlow(creds, loopback)
}

func (g *GoogleFlow) AuthURL() string {
	return g.cfg.AuthCodeURL(g.callback.State(),
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.S256ChallengeOption(g.callback.Verifier()),
	)
}

func (g *GoogleFlow) Wait(ctx context.Context, timeout time.Duration) (*oauth2.Token, error) {
	defer g.callback.Close()

	res, err := g.callback.Wait(ctx, timeout)
	if err != nil {
		return nil, err
	}

	tok, err := g.cfg.Exchange(ctx, res.Code, oauth2.VerifierOption(g.callback.Verifier()))
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return tok, nil
}

func (g *GoogleFlow) Config() *oauth2.Config            { return g.cfg }
func (g *GoogleFlow) Credentials() GoogleAppCredentials { return g.creds }
func (g *GoogleFlow) State() string                     { return g.callback.State() }

func (g *GoogleFlow) Close() error { return g.callback.Close() }

func MarshalToken(tok *oauth2.Token) ([]byte, error) { return json.Marshal(tok) }

func UnmarshalToken(data []byte) (*oauth2.Token, error) {
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}
