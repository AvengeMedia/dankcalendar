package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Work with calendar events",
	Args:  cobra.NoArgs,
}

var eventsRSVPCmd = &cobra.Command{
	Use:   "rsvp <event-id> <accept|decline|tentative>",
	Short: "Respond to a meeting invitation",
	Long:  "Submit your accept, decline, or tentative reply to a synced meeting invitation and push it back to the provider.",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		result, err := remindersCall("events.rsvp", map[string]any{"id": args[0], "response": args[1]})
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(result)
		}
		fmt.Printf("RSVP sent: %s\n", args[1])
		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsRSVPCmd)
}
