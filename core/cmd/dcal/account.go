package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/internal/accounts"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
	dankkeyring "github.com/AvengeMedia/dankcalendar/core/internal/keyring"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage calendar accounts",
}

var accountRemoveYes bool

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured accounts",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		items, err := st.repo.ListAccounts(ctx)
		if err != nil {
			return err
		}

		if jsonOutput {
			type accountOut struct {
				ID          string         `json:"id"`
				Kind        string         `json:"kind"`
				Provider    string         `json:"provider"`
				DisplayName string         `json:"displayName"`
				Authorized  bool           `json:"authorized"`
				NeedsReauth bool           `json:"needsReauth"`
				Settings    map[string]any `json:"settings,omitempty"`
			}
			out := make([]accountOut, 0, len(items))
			for _, a := range items {
				out = append(out, accountOut{
					ID:          a.ID,
					Kind:        string(a.Kind),
					Provider:    accounts.Flavor(string(a.Kind), a.Settings),
					DisplayName: a.DisplayName,
					Authorized:  accounts.Authorized(ctx, st.secrets, a),
					NeedsReauth: a.NeedsReauth,
					Settings:    a.Settings,
				})
			}
			return printJSON(out)
		}

		if len(items) == 0 {
			fmt.Println("No accounts configured. Run `dcal account add --help` to see providers.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tPROVIDER\tNAME\tSTATUS")
		for _, a := range items {
			status := "ok"
			switch {
			case a.NeedsReauth:
				status = "needs reauth"
			case !accounts.Authorized(ctx, st.secrets, a):
				status = "needs auth"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, accounts.Flavor(string(a.Kind), a.Settings), a.DisplayName, status)
		}
		return w.Flush()
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:     "remove <account-id>",
	Aliases: []string{"delete"},
	Short:   "Remove an account and all of its locally synced data",
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		acc, err := st.repo.GetAccount(ctx, args[0])
		if err != nil {
			if repo.IsNotFound(err) {
				return fmt.Errorf("no account %q (see `dcal account list`)", args[0])
			}
			return err
		}

		if !accountRemoveYes {
			provider := accounts.Flavor(string(acc.Kind), acc.Settings)
			ok, err := confirm(fmt.Sprintf("Remove %s [%s] and all of its locally synced data? [y/N] ", acc.ID, provider))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}

		if err := accounts.Delete(ctx, st.repo, st.secrets, acc.ID); err != nil {
			return err
		}
		notifyDaemon(acc.ID, false)
		if jsonOutput {
			return printJSON(map[string]any{"accountId": acc.ID, "removed": true})
		}
		fmt.Printf("removed account %s\n", acc.ID)
		return nil
	},
}

var accountProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List supported account providers",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		providers := accounts.AvailableProviders()
		if jsonOutput {
			return printJSON(providers)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
		for _, p := range providers {
			fmt.Fprintf(w, "%s\t%s\t%s\n", p.ID, p.Name, p.Description)
		}
		return w.Flush()
	},
}

var accountSetupCmd = &cobra.Command{
	Use:       "setup <google|microsoft>",
	Short:     "Print instructions for creating your own OAuth app (optional for Google)",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"google", "microsoft"},
	RunE: func(_ *cobra.Command, args []string) error {
		var steps []accounts.SetupStep
		switch args[0] {
		case "google":
			steps = accounts.GoogleSetupSteps()
		case "microsoft":
			steps = accounts.MicrosoftSetupSteps()
		default:
			return fmt.Errorf("unknown provider %q (expected google or microsoft)", args[0])
		}

		if jsonOutput {
			return printJSON(steps)
		}

		builtin := false
		switch args[0] {
		case "google":
			_, builtin = oauth.BuiltinGoogleCredentials()
		case "microsoft":
			_, builtin = oauth.BuiltinMicrosoftCredentials()
		}
		if builtin {
			fmt.Printf("Dank Calendar ships with a built-in %s client — `dcal account add %s` just works.\n", accounts.ProviderName(args[0]), args[0])
			fmt.Println("Follow these steps only if you want to use your own OAuth client.")
			fmt.Println()
		}

		fmt.Printf("%s setup (one-time)\n\n", accounts.ProviderName(args[0]))
		for i, s := range steps {
			fmt.Printf("%d. %s\n", i+1, s.Title)
			fmt.Printf("   %s\n", s.Description)
			if s.Note != "" {
				fmt.Printf("   Note: %s\n", s.Note)
			}
			if s.URL != "" {
				fmt.Printf("   %s\n", s.URL)
			}
			fmt.Println()
		}
		switch args[0] {
		case "google":
			fmt.Println("Then run:  dcal account add google --client-id <id> --client-secret <secret>")
			fmt.Println("      or:  dcal account add google --credentials client_secret.json")
		case "microsoft":
			fmt.Println("Then run:  dcal account add microsoft --client-id <id>")
		default:
			fmt.Printf("Then run:  dcal account add %s\n", args[0])
		}
		return nil
	},
}

func init() {
	accountRemoveCmd.Flags().BoolVarP(&accountRemoveYes, "yes", "y", false, "Skip the confirmation prompt")

	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountReauthCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountProvidersCmd)
	accountCmd.AddCommand(accountSetupCmd)
}

type cliStores struct {
	repo    *repo.Repo
	secrets calendar.SecretStore
}

// openStores gives the CLI the same storage the daemon uses: the sqlite
// database plus the keyring-backed secret store, so both can run side by side.
func openStores(ctx context.Context) (*cliStores, func(), error) {
	cfg := config.New()
	dbPath := cfg.DatabasePath
	if dbPath == "" {
		path, err := paths.DatabasePath()
		if err != nil {
			return nil, nil, err
		}
		dbPath = path
	}

	client, err := repo.OpenFile(ctx, dbPath)
	if err != nil {
		return nil, nil, err
	}
	r := repo.New(client)
	secrets := dankkeyring.NewSecretStore(dankkeyring.Open(), repo.NewSecretStore(r))
	return &cliStores{repo: r, secrets: secrets}, func() { _ = r.Close() }, nil
}

// notifyDaemon tells a running daemon (if any) that accounts changed so the
// GUI updates, optionally asking it to sync the account. Best-effort.
func notifyDaemon(accountID string, refresh bool) bool {
	socketPath, err := ipc.FindRunningSocket()
	if err != nil {
		return false
	}
	client, err := ipc.Dial(socketPath)
	if err != nil {
		return false
	}
	defer client.Close()

	params := map[string]any{"accountId": accountID}
	resp, err := client.Call(ipc.Request{ID: 1, Method: "accounts.changed", Params: params})
	if err != nil || resp.Error != "" {
		return false
	}
	if !refresh {
		return true
	}
	resp, err = client.Call(ipc.Request{ID: 2, Method: "accounts.refresh", Params: params})
	return err == nil && resp.Error == ""
}

func confirm(label string) (bool, error) {
	answer, err := readPrompt(bufio.NewReader(os.Stdin), label)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(answer) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

// Prompt labels go to stderr so stdout stays clean for piping and --json.
func readPrompt(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readPassword(label string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return readPrompt(bufio.NewReader(os.Stdin), label)
	}

	fmt.Fprint(os.Stderr, label)
	data, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func oauthContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Minute)
}
