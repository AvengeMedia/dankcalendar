package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/caldav"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/evolution"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/google"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/ical"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/local"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/microsoft"
	"github.com/AvengeMedia/dankcalendar/core/internal/sync"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

var syncCmd = &cobra.Command{
	Use:   "sync [account-id]",
	Short: "Sync all configured accounts, or a single account",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		accountID := ""
		if len(args) == 1 {
			accountID = args[0]
		}

		if notifyDaemon(accountID, true) {
			if jsonOutput {
				return printJSON(syncResult("started", true, accountID))
			}
			fmt.Println("sync started in the running dcal daemon")
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		st, closer, err := openStores(ctx)
		if err != nil {
			return err
		}
		defer closer()

		switch accountID {
		case "":
			if err := sync.NewEngine(st.repo, providerRegistry(), st.secrets, time.Hour).SyncAll(ctx); err != nil {
				return err
			}
		default:
			if err := syncOneAccount(ctx, st, accountID); err != nil {
				if repo.IsNotFound(err) {
					return fmt.Errorf("no account %q (see `dcal account list`)", accountID)
				}
				return err
			}
		}

		if jsonOutput {
			return printJSON(syncResult("complete", false, accountID))
		}
		log.Info("sync complete")
		return nil
	},
}

func syncResult(status string, daemon bool, accountID string) map[string]any {
	out := map[string]any{"status": status, "daemon": daemon}
	if accountID != "" {
		out["accountId"] = accountID
	}
	return out
}

func providerRegistry() *calendar.Registry {
	registry := calendar.NewRegistry()
	registry.Register(local.Factory{})
	registry.Register(google.Factory{})
	registry.Register(caldav.Factory{})
	registry.Register(microsoft.Factory{})
	registry.Register(ical.Factory{})
	registry.Register(evolution.Factory{})
	return registry
}

func syncOneAccount(ctx context.Context, st *cliStores, accountID string) error {
	acc, err := st.repo.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}
	engine := sync.NewEngine(st.repo, providerRegistry(), st.secrets, time.Hour)
	return engine.SyncAccount(ctx, acc)
}
