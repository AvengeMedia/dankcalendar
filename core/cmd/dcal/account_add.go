package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/internal/accounts"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

var accountAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a calendar account (google, microsoft, caldav, icloud, ical, local)",
}

var accountAddLocalCmd = &cobra.Command{
	Use:   "local <dir>",
	Short: "Add a local ICS account (a directory of .ics files)",
	Long:  "Point at a directory of .ics files. Each subdirectory or .ics file becomes a calendar — compatible with vdirsyncer/khal layouts.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")

		ctx := context.Background()
		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := accounts.AddLocal(ctx, st.repo, accounts.LocalInput{Root: root, DisplayName: name})
		if err != nil {
			return err
		}
		return finishAdd(ctx, st, res)
	},
}

var accountAddCalDAVCmd = &cobra.Command{
	Use:   "caldav",
	Short: "Add a CalDAV account (any CalDAV-compatible server)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAddCalDAV(cmd, false)
	},
}

var accountAddICloudCmd = &cobra.Command{
	Use:   "icloud",
	Short: "Add an Apple iCloud account (app-specific password)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAddCalDAV(cmd, true)
	},
}

func runAddCalDAV(cmd *cobra.Command, icloud bool) error {
	serverURL, _ := cmd.Flags().GetString("url")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	name, _ := cmd.Flags().GetString("name")
	if icloud {
		serverURL = accounts.ICloudCalDAVURL
		infof("iCloud requires an app-specific password — regular Apple ID passwords will not work.")
		infof("Create one at https://account.apple.com/account/manage")
		infof("")
	}

	reader := bufio.NewReader(os.Stdin)
	var err error
	if serverURL == "" {
		serverURL, err = readPrompt(reader, "Server URL (e.g. https://dav.example.com): ")
		if err != nil {
			return err
		}
	}
	if username == "" {
		label := "Username: "
		if icloud {
			label = "Apple ID: "
		}
		username, err = readPrompt(reader, label)
		if err != nil {
			return err
		}
	}
	if password == "" {
		label := "Password: "
		if icloud {
			label = "App-specific password (xxxx-xxxx-xxxx-xxxx): "
		}
		password, err = readPassword(label)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	st, closer, err := openStores(ctx)
	if err != nil {
		return err
	}
	defer closer()

	infof("verifying connection...")
	res, err := accounts.AddCalDAV(ctx, st.repo, st.secrets, accounts.CalDAVInput{
		URL:         serverURL,
		Username:    username,
		Password:    password,
		DisplayName: name,
	})
	if err != nil {
		return err
	}
	return finishAdd(ctx, st, res)
}

var accountAddICalCmd = &cobra.Command{
	Use:   "ical <url>",
	Short: "Subscribe to an iCal feed by URL (webcal or https)",
	Long:  "Subscribe to a read-only external calendar feed, e.g. a university timetable or a published Google/Outlook calendar. webcal:// URLs are accepted.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		name, _ := cmd.Flags().GetString("name")

		if username != "" && password == "" {
			var err error
			password, err = readPassword("Password: ")
			if err != nil {
				return err
			}
		}

		ctx := context.Background()
		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		infof("fetching feed...")
		res, err := accounts.AddICal(ctx, st.repo, st.secrets, accounts.ICalInput{
			URL:         args[0],
			Username:    username,
			Password:    password,
			DisplayName: name,
		})
		if err != nil {
			return err
		}
		return finishAdd(ctx, st, res)
	},
}

var accountAddGoogleCmd = &cobra.Command{
	Use:   "google",
	Short: "Add or re-authorize a Google account (OAuth in your browser)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		creds, err := resolveGoogleCreds(cmd)
		if err != nil {
			return err
		}

		ctx, cancel := oauthContext()
		defer cancel()

		flow, err := oauth.StartGoogleLoopbackFlow(ctx, creds, config.New().OAuthBindAddr)
		if err != nil {
			return err
		}

		tok, err := waitForBrowserAuth(ctx, flow.AuthURL(), flow.Config().RedirectURL, flow.Wait)
		if err != nil {
			return err
		}

		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := accounts.FinishGoogle(ctx, st.repo, st.secrets, flow, tok)
		if err != nil {
			return err
		}
		return finishAdd(ctx, st, res)
	},
}

var accountAddMicrosoftCmd = &cobra.Command{
	Use:   "microsoft",
	Short: "Add or re-authorize a Microsoft account (OAuth in your browser)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		clientID, _ := cmd.Flags().GetString("client-id")
		tenant, _ := cmd.Flags().GetString("tenant")

		if clientID == "" {
			infof("A Microsoft app registration is required.")
			infof("Run `dcal account setup microsoft` for step-by-step instructions.")
			infof("")
			var err error
			clientID, err = readPrompt(bufio.NewReader(os.Stdin), "Application (client) ID: ")
			if err != nil {
				return err
			}
		}

		ctx, cancel := oauthContext()
		defer cancel()

		creds := oauth.MicrosoftAppCredentials{ClientID: clientID, Tenant: tenant}
		flow, err := oauth.StartMicrosoftLoopbackFlow(ctx, creds, config.New().OAuthBindAddr)
		if err != nil {
			return err
		}

		tok, err := waitForBrowserAuth(ctx, flow.AuthURL(), flow.Config().RedirectURL, flow.Wait)
		if err != nil {
			return err
		}

		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := accounts.FinishMicrosoft(ctx, st.repo, st.secrets, flow, tok)
		if err != nil {
			return err
		}
		return finishAdd(ctx, st, res)
	},
}

func init() {
	accountAddLocalCmd.Flags().String("name", "", "Display name (defaults to the directory name)")

	for _, c := range []*cobra.Command{accountAddCalDAVCmd, accountAddICloudCmd} {
		c.Flags().String("username", "", "Username")
		c.Flags().String("password", "", "Password (prompted securely if omitted)")
		c.Flags().String("name", "", "Display name")
	}
	accountAddCalDAVCmd.Flags().String("url", "", "Server URL (e.g. https://dav.example.com)")

	accountAddICalCmd.Flags().String("username", "", "Basic-auth username (optional)")
	accountAddICalCmd.Flags().String("password", "", "Basic-auth password (prompted if a username is given)")
	accountAddICalCmd.Flags().String("name", "", "Display name (defaults to the feed name)")

	accountAddGoogleCmd.Flags().String("client-id", "", "OAuth client ID")
	accountAddGoogleCmd.Flags().String("client-secret", "", "OAuth client secret")
	accountAddGoogleCmd.Flags().String("credentials", "", "Path to the client_secret_….json downloaded from Google")

	accountAddMicrosoftCmd.Flags().String("client-id", "", "Application (client) ID from your app registration")
	accountAddMicrosoftCmd.Flags().String("tenant", "", `Entra tenant (defaults to "common")`)

	accountAddCmd.AddCommand(accountAddLocalCmd)
	accountAddCmd.AddCommand(accountAddCalDAVCmd)
	accountAddCmd.AddCommand(accountAddICloudCmd)
	accountAddCmd.AddCommand(accountAddICalCmd)
	accountAddCmd.AddCommand(accountAddGoogleCmd)
	accountAddCmd.AddCommand(accountAddMicrosoftCmd)
}

func resolveGoogleCreds(cmd *cobra.Command) (oauth.GoogleAppCredentials, error) {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")
	credsFile, _ := cmd.Flags().GetString("credentials")

	creds := oauth.GoogleAppCredentials{ClientID: clientID, ClientSecret: clientSecret}

	if credsFile != "" {
		fileCreds, err := readGoogleCredentialsFile(credsFile)
		if err != nil {
			return creds, err
		}
		if creds.ClientID == "" {
			creds.ClientID = fileCreds.ClientID
		}
		if creds.ClientSecret == "" {
			creds.ClientSecret = fileCreds.ClientSecret
		}
	}

	cfg := config.New()
	if creds.ClientID == "" {
		creds.ClientID = cfg.GoogleClientID
	}
	if creds.ClientSecret == "" {
		creds.ClientSecret = cfg.GoogleSecret
	}
	if creds.ClientID != "" && creds.ClientSecret != "" {
		return creds, nil
	}

	infof("Google OAuth client credentials are required.")
	infof("Run `dcal account setup google` for step-by-step instructions.")
	infof("")

	reader := bufio.NewReader(os.Stdin)
	var err error
	if creds.ClientID == "" {
		creds.ClientID, err = readPrompt(reader, "Client ID: ")
		if err != nil {
			return creds, err
		}
	}
	if creds.ClientSecret == "" {
		creds.ClientSecret, err = readPrompt(reader, "Client Secret: ")
		if err != nil {
			return creds, err
		}
	}
	return creds, nil
}

func readGoogleCredentialsFile(path string) (oauth.GoogleAppCredentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return oauth.GoogleAppCredentials{}, err
	}

	var doc struct {
		Installed *oauth.GoogleAppCredentials `json:"installed"`
		Web       *oauth.GoogleAppCredentials `json:"web"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return oauth.GoogleAppCredentials{}, fmt.Errorf("parse %s: %w", path, err)
	}

	client := doc.Installed
	if client == nil {
		client = doc.Web
	}
	if client == nil || client.ClientID == "" {
		return oauth.GoogleAppCredentials{}, fmt.Errorf("no OAuth client found in %s (expected the client_secret_….json downloaded from Google)", path)
	}
	return *client, nil
}

func waitForBrowserAuth(ctx context.Context, authURL, redirectURL string, wait func(context.Context, time.Duration) (*oauth2.Token, error)) (*oauth2.Token, error) {
	openBrowser(authURL)
	infof("")
	infof("Open the following URL in your browser to authorize Dank Calendar:")
	infof("")
	infof("    %s", authURL)
	infof("")
	infof("waiting for the browser redirect on %s ...", redirectURL)

	return wait(ctx, 5*time.Minute)
}

func openBrowser(url string) {
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }()
}

func finishAdd(ctx context.Context, st *cliStores, res accounts.Result) error {
	infof("account %s (%s) configured", res.AccountID, res.DisplayName)

	syncStatus := "complete"
	var syncErr error
	switch {
	case notifyDaemon(res.AccountID, true):
		syncStatus = "daemon"
		infof("syncing in the running dcal daemon")
	default:
		infof("syncing...")
		if syncErr = syncOneAccount(ctx, st, res.AccountID); syncErr != nil {
			syncStatus = "failed"
		} else {
			infof("sync complete")
		}
	}

	if jsonOutput {
		provider := ""
		if acc, err := st.repo.GetAccount(ctx, res.AccountID); err == nil {
			provider = accounts.Flavor(string(acc.Kind), acc.Settings)
		}
		if err := printJSON(map[string]any{
			"accountId":   res.AccountID,
			"displayName": res.DisplayName,
			"provider":    provider,
			"sync":        syncStatus,
		}); err != nil {
			return err
		}
	}

	if syncErr != nil {
		return fmt.Errorf("initial sync failed (run `dcal sync %s` to retry): %w", res.AccountID, syncErr)
	}
	return nil
}
