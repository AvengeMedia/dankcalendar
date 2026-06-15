package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
)

var ipcCmd = &cobra.Command{
	Use:   "ipc <method> [key=value...]",
	Short: "Send an IPC request to the running dcal daemon",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		method := args[0]
		params, err := parseParams(args[1:])
		if err != nil {
			return err
		}

		socketPath := os.Getenv("DANKCAL_SOCKET")
		if socketPath == "" {
			socketPath, err = ipc.FindRunningSocket()
			if err != nil {
				return fmt.Errorf("dcal daemon not running: %w", err)
			}
		}

		client, err := ipc.Dial(socketPath)
		if err != nil {
			return err
		}
		defer client.Close()

		resp, err := client.Call(ipc.Request{ID: 1, Method: method, Params: params})
		if err != nil {
			return err
		}

		if resp.Error != "" {
			return fmt.Errorf("ipc error: %s", resp.Error)
		}

		out, err := json.MarshalIndent(resp.Result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(out))
		return nil
	},
}

func parseParams(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, nil
	}
	params := make(map[string]any, len(args))
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("expected key=value, got %q", arg)
		}
		params[key] = value
	}
	return params, nil
}
