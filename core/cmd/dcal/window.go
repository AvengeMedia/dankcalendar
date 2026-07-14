package main

import (
	"github.com/spf13/cobra"
)

var windowView string

var showCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show the calendar window, launching dcal if it is not running",
	PreRunE: shellApp.ResolveConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return shellApp.CallOrLaunch("ui.show", viewParams())
	},
}

var toggleCmd = &cobra.Command{
	Use:     "toggle",
	Short:   "Toggle calendar window visibility, launching dcal if it is not running",
	PreRunE: shellApp.ResolveConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return shellApp.CallOrLaunch("ui.toggle", viewParams())
	},
}

var openCmd = &cobra.Command{
	Use:     "open [url]",
	Short:   "Open a webcal:// subscription link (no url just shows the window)",
	PreRunE: shellApp.ResolveConfig,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "" {
			return shellApp.CallOrLaunch("ui.show", nil)
		}
		return shellApp.CallOrLaunch("ui.open", map[string]any{"url": args[0]})
	},
}

func viewParams() map[string]any {
	if windowView == "" {
		return nil
	}
	return map[string]any{"view": windowView}
}
