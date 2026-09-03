package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Work with calendar events",
	Args:  cobra.NoArgs,
}

var eventsRSVPOccurrenceStart string

var eventsRSVPCmd = &cobra.Command{
	Use:   "rsvp <event-id> <accept|decline|tentative>",
	Short: "Respond to a meeting invitation",
	Long:  "Submit your accept, decline, or tentative reply to a synced meeting invitation and push it back to the provider.",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		params := map[string]any{"id": args[0], "response": args[1]}
		if eventsRSVPOccurrenceStart != "" {
			params["occurrenceStart"] = eventsRSVPOccurrenceStart
		}
		result, err := remindersCall("events.rsvp", params)
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

var eventsImportCalendar string

var eventsImportCmd = &cobra.Command{
	Use:   "import <file.ics>",
	Short: "Import events from an .ics file into a calendar",
	Long:  "Import every event in an .ics file (e.g. an emailed invitation) into the given calendar. Events already present in that calendar are left untouched. Find calendar ids with 'dcal ipc calendars.list'.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if eventsImportCalendar == "" {
			return errors.New("--calendar is required (ids: dcal ipc calendars.list)")
		}
		ics, err := readICSFile(args[0])
		if err != nil {
			return err
		}
		result, err := remindersCall("events.importIcs", map[string]any{"ics": ics, "calendarId": eventsImportCalendar})
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(result)
		}
		items, err := decodeImported(result)
		if err != nil {
			return err
		}
		for _, item := range items {
			state := "imported"
			if item.Existing {
				state = "already present"
			}
			fmt.Printf("%s  %s  (%s)\n", item.Event.Start.Local().Format("Mon Jan 2 15:04"), item.Event.Summary, state)
		}
		return nil
	},
}

type importedItem struct {
	Event struct {
		ID      string    `json:"id"`
		Summary string    `json:"summary"`
		Start   time.Time `json:"start"`
	} `json:"event"`
	Existing bool `json:"existing"`
}

func decodeImported(result any) ([]importedItem, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Events []importedItem `json:"events"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Events, nil
}

func init() {
	eventsRSVPCmd.Flags().StringVar(&eventsRSVPOccurrenceStart, "occurrence-start", "", "RFC3339 start time; reply for this single occurrence of a recurring event")
	eventsImportCmd.Flags().StringVar(&eventsImportCalendar, "calendar", "", "Target calendar id (see 'dcal ipc calendars.list')")
	eventsCmd.AddCommand(eventsRSVPCmd)
	eventsCmd.AddCommand(eventsImportCmd)
}
