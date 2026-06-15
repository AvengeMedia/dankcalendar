package accounts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	googleoauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	caldavprovider "github.com/AvengeMedia/dankcalendar/core/internal/providers/caldav"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/google"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/microsoft"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type Result struct {
	AccountID   string
	DisplayName string
}

type CalDAVInput struct {
	URL         string
	Username    string
	Password    string
	DisplayName string
}

// AddCalDAV verifies the connection before the account is saved. iCloud is a
// caldav account pointed at ICloudCalDAVURL with the "icloud" preset marker.
func AddCalDAV(ctx context.Context, r *repo.Repo, secrets calendar.SecretStore, in CalDAVInput) (Result, error) {
	serverURL := strings.TrimSpace(in.URL)
	username := strings.TrimSpace(in.Username)
	displayName := strings.TrimSpace(in.DisplayName)

	switch {
	case serverURL == "":
		return Result{}, errors.New("url is required")
	case username == "":
		return Result{}, errors.New("username is required")
	case in.Password == "":
		return Result{}, errors.New("password is required")
	}

	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Host == "" {
		return Result{}, errors.New("url is not a valid URL (expected e.g. https://dav.example.com)")
	}

	accountID := truncateID(fmt.Sprintf("%s@%s", username, parsed.Host))
	if displayName == "" {
		displayName = username
	}

	settings := map[string]any{"url": serverURL, "username": username}
	if parsed.Host == "caldav.icloud.com" || strings.HasSuffix(parsed.Host, ".icloud.com") {
		settings["preset"] = "icloud"
	}

	domainAcc := calendar.Account{ID: accountID, Kind: calendar.AccountCalDAV, Settings: settings}
	probe := memSecrets{caldavprovider.SecretKeyPassword: []byte(in.Password)}

	provider, err := caldavprovider.Factory{}.Build(ctx, domainAcc, probe)
	if err != nil {
		return Result{}, fmt.Errorf("caldav connection failed: %w", err)
	}
	defer provider.Close()
	if _, err := provider.ListCalendars(ctx); err != nil {
		return Result{}, fmt.Errorf("caldav connection failed: %w", err)
	}

	if err := Ensure(ctx, r, accountID, account.KindCaldav, displayName, settings); err != nil {
		return Result{}, err
	}
	if err := secrets.Set(ctx, accountID, caldavprovider.SecretKeyPassword, []byte(in.Password)); err != nil {
		return Result{}, err
	}
	return Result{AccountID: accountID, DisplayName: displayName}, nil
}

type LocalInput struct {
	Root        string
	DisplayName string
}

func AddLocal(ctx context.Context, r *repo.Repo, in LocalInput) (Result, error) {
	root := strings.TrimSpace(in.Root)
	displayName := strings.TrimSpace(in.DisplayName)

	switch {
	case root == "":
		return Result{}, errors.New("root is required (path to a directory of .ics files)")
	case !filepath.IsAbs(root):
		return Result{}, errors.New("root must be an absolute path")
	}

	accountID := truncateID("local:" + filepath.Base(root))
	if displayName == "" {
		displayName = filepath.Base(root)
	}

	settings := map[string]any{"root": root}
	if err := Ensure(ctx, r, accountID, account.KindLocal, displayName, settings); err != nil {
		return Result{}, err
	}
	return Result{AccountID: accountID, DisplayName: displayName}, nil
}

// FinishGoogle stores a completed OAuth flow as an account keyed by the
// authorized email address.
func FinishGoogle(ctx context.Context, r *repo.Repo, secrets calendar.SecretStore, flow *oauth.GoogleFlow, tok *oauth2.Token) (Result, error) {
	email, err := FetchGoogleEmail(ctx, flow.Config(), tok)
	if err != nil {
		return Result{}, fmt.Errorf("fetch google profile: %w", err)
	}

	accountID := strings.ToLower(email)
	if accountID == "" {
		return Result{}, errors.New("google account did not return an email address")
	}

	if err := Ensure(ctx, r, accountID, account.KindGoogle, email, nil); err != nil {
		return Result{}, err
	}

	appBytes, _ := json.Marshal(flow.Credentials())
	if err := secrets.Set(ctx, accountID, google.SecretKeyApp, appBytes); err != nil {
		return Result{}, err
	}
	tokenBytes, _ := oauth.MarshalToken(tok)
	if err := secrets.Set(ctx, accountID, google.SecretKeyToken, tokenBytes); err != nil {
		return Result{}, err
	}
	if err := r.SetAccountAuthState(ctx, accountID, false, ""); err != nil {
		return Result{}, err
	}
	return Result{AccountID: accountID, DisplayName: email}, nil
}

func FinishMicrosoft(ctx context.Context, r *repo.Repo, secrets calendar.SecretStore, flow *oauth.MicrosoftFlow, tok *oauth2.Token) (Result, error) {
	email, err := FetchMicrosoftEmail(ctx, flow.Config(), tok)
	if err != nil {
		return Result{}, fmt.Errorf("fetch microsoft profile: %w", err)
	}

	accountID := strings.ToLower(email)
	if accountID == "" {
		return Result{}, errors.New("microsoft account did not return an email address")
	}

	if err := Ensure(ctx, r, accountID, account.KindMicrosoft, email, nil); err != nil {
		return Result{}, err
	}

	appBytes, _ := json.Marshal(flow.Credentials())
	if err := secrets.Set(ctx, accountID, microsoft.SecretKeyApp, appBytes); err != nil {
		return Result{}, err
	}
	tokenBytes, _ := oauth.MarshalToken(tok)
	if err := secrets.Set(ctx, accountID, microsoft.SecretKeyToken, tokenBytes); err != nil {
		return Result{}, err
	}
	if err := r.SetAccountAuthState(ctx, accountID, false, ""); err != nil {
		return Result{}, err
	}
	return Result{AccountID: accountID, DisplayName: email}, nil
}

func FetchGoogleEmail(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token) (string, error) {
	svc, err := googleoauth2.NewService(ctx, option.WithTokenSource(cfg.TokenSource(ctx, tok)))
	if err != nil {
		return "", err
	}
	info, err := svc.Userinfo.Get().Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return info.Email, nil
}

func FetchMicrosoftEmail(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token) (string, error) {
	client := oauth2.NewClient(ctx, cfg.TokenSource(ctx, tok))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("graph /me returned %s", resp.Status)
	}

	var me struct {
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return "", err
	}
	switch {
	case me.Mail != "":
		return me.Mail, nil
	case me.UserPrincipalName != "":
		return me.UserPrincipalName, nil
	}
	return "", errors.New("graph profile has no email")
}

type memSecrets map[string][]byte

func (m memSecrets) Get(_ context.Context, _ string, key string) ([]byte, error) {
	v, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("secret %q not found", key)
	}
	return v, nil
}

func (m memSecrets) Set(_ context.Context, _ string, key string, value []byte) error {
	m[key] = value
	return nil
}

func (m memSecrets) Delete(_ context.Context, _ string, key string) error {
	delete(m, key)
	return nil
}
