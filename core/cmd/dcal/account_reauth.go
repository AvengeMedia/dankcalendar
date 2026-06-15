package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/internal/accounts"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

var accountReauthCmd = &cobra.Command{
	Use:   "reauth <account-id>",
	Short: "Re-authorize a Google or Microsoft account whose login expired",
	Long:  "Runs the OAuth browser flow again for an existing account, reusing the app credentials it was added with. Use this when an account shows \"needs reauth\".",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		ctx, cancel := oauthContext()
		defer cancel()

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

		var res accounts.Result
		switch acc.Kind {
		case account.KindGoogle:
			res, err = reauthGoogle(ctx, st, acc.ID)
		case account.KindMicrosoft:
			res, err = reauthMicrosoft(ctx, st, acc.ID)
		default:
			return fmt.Errorf("account %q is not an OAuth account; re-add it with `dcal account add`", acc.ID)
		}
		if err != nil {
			return err
		}
		return finishAdd(ctx, st, res)
	},
}

func reauthGoogle(ctx context.Context, st *cliStores, accountID string) (accounts.Result, error) {
	creds, err := accounts.GoogleAppCreds(ctx, st.secrets, accountID)
	if err != nil {
		return accounts.Result{}, err
	}

	flow, err := oauth.StartGoogleLoopbackFlow(ctx, creds, config.New().OAuthBindAddr)
	if err != nil {
		return accounts.Result{}, err
	}

	tok, err := waitForBrowserAuth(ctx, flow.AuthURL(), flow.Config().RedirectURL, flow.Wait)
	if err != nil {
		return accounts.Result{}, err
	}
	return accounts.FinishGoogle(ctx, st.repo, st.secrets, flow, tok)
}

func reauthMicrosoft(ctx context.Context, st *cliStores, accountID string) (accounts.Result, error) {
	creds, err := accounts.MicrosoftAppCreds(ctx, st.secrets, accountID)
	if err != nil {
		return accounts.Result{}, err
	}

	flow, err := oauth.StartMicrosoftLoopbackFlow(ctx, creds, config.New().OAuthBindAddr)
	if err != nil {
		return accounts.Result{}, err
	}

	tok, err := waitForBrowserAuth(ctx, flow.AuthURL(), flow.Config().RedirectURL, flow.Wait)
	if err != nil {
		return accounts.Result{}, err
	}
	return accounts.FinishMicrosoft(ctx, st.repo, st.secrets, flow, tok)
}
