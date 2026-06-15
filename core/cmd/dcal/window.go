package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
)

var showCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show the calendar window, launching dcal if it is not running",
	PreRunE: findConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return showOrLaunch("show")
	},
}

var toggleCmd = &cobra.Command{
	Use:     "toggle",
	Short:   "Toggle calendar window visibility, launching dcal if it is not running",
	PreRunE: findConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return showOrLaunch("toggle")
	},
}

func showOrLaunch(action string) error {
	if err := sendUIAction(action); err == nil {
		return nil
	}
	return runShellDaemon(false)
}

func sendUIAction(action string) error {
	socketPath, err := ipc.FindRunningSocket()
	if err != nil {
		return err
	}

	client, err := ipc.Dial(socketPath)
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Call(ipc.Request{ID: 1, Method: "ui." + action})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}
