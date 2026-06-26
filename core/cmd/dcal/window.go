package main

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
)

var windowView string

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

var openCmd = &cobra.Command{
	Use:     "open [url]",
	Short:   "Open a webcal:// subscription link (no url just shows the window)",
	PreRunE: findConfig,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "" {
			return showOrLaunch("show")
		}
		return openLink(args[0])
	},
}

func showOrLaunch(action string) error {
	if err := sendUIAction(action, windowView); err == nil {
		return nil
	}
	if err := runShellDaemon(false); err != nil {
		return err
	}
	if windowView == "" {
		return nil
	}

	// A fresh window opens on its persisted view, so retry the action until the
	// daemon's socket is up to honor the requested view on cold start.
	deadline := time.Now().Add(15 * time.Second)
	for {
		if err := sendUIAction(action, windowView); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for dcal to start")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// openLink hands a subscription URL to a running instance, or cold-starts one
// and retries until its socket is up; the daemon holds the URL until the GUI
// subscribes.
func openLink(url string) error {
	if err := sendUIOpen(url); err == nil {
		return nil
	}
	if err := runShellDaemon(false); err != nil {
		return err
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		if err := sendUIOpen(url); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for dcal to start")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func sendUIOpen(url string) error {
	socketPath, err := ipc.FindRunningSocket()
	if err != nil {
		return err
	}

	client, err := ipc.Dial(socketPath)
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Call(ipc.Request{ID: 1, Method: "ui.open", Params: map[string]any{"url": url}})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func sendUIAction(action, view string) error {
	socketPath, err := ipc.FindRunningSocket()
	if err != nil {
		return err
	}

	client, err := ipc.Dial(socketPath)
	if err != nil {
		return err
	}
	defer client.Close()

	req := ipc.Request{ID: 1, Method: "ui." + action}
	if view != "" {
		req.Params = map[string]any{"view": view}
	}

	resp, err := client.Call(req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}
