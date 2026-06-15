package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
	"github.com/AvengeMedia/dankcalendar/core/internal/reminders"
)

var remindersLimit int

var remindersCmd = &cobra.Command{
	Use:   "reminders",
	Short: "List upcoming reminders scheduled by the running daemon",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		result, err := remindersCall("reminders.upcoming", map[string]any{"limit": remindersLimit})
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(result)
		}

		items, err := decodeUpcoming(result)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("no upcoming reminders")
			return nil
		}

		for _, item := range items {
			line := fmt.Sprintf("%s  %s", item.Trigger.Local().Format("Mon Jan 2 15:04"), item.Summary)
			switch {
			case item.Snoozed:
				line += "  (snoozed)"
			case item.AllDay:
				line += "  (all day)"
			default:
				line += fmt.Sprintf("  (starts %s)", item.Start.Local().Format("15:04"))
			}
			fmt.Println(line)
		}
		return nil
	},
}

var remindersTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test desktop notification through the daemon",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		result, err := remindersCall("reminders.test", nil)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(result)
		}
		fmt.Println("test notification sent")
		return nil
	},
}

func init() {
	remindersCmd.Flags().IntVar(&remindersLimit, "limit", 20, "Maximum number of reminders to list")
	remindersCmd.AddCommand(remindersTestCmd)
}

func remindersCall(method string, params map[string]any) (any, error) {
	socketPath := os.Getenv("DANKCAL_SOCKET")
	if socketPath == "" {
		path, err := ipc.FindRunningSocket()
		if err != nil {
			return nil, fmt.Errorf("dcal daemon not running: %w", err)
		}
		socketPath = path
	}

	client, err := ipc.Dial(socketPath)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	resp, err := client.Call(ipc.Request{ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Result, nil
}

func decodeUpcoming(result any) ([]reminders.Upcoming, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Reminders []reminders.Upcoming `json:"reminders"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Reminders, nil
}
