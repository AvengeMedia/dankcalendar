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
	Use:     "open [url|file.ics|file.vcs]...",
	Short:   "Open a webcal:// subscription link or an .ics or .vcs file (no argument just shows the window)",
	Long:    "Open a webcal:// subscription link in the UI, or an .ics or .vcs file (e.g. an emailed invitation) in the import dialog. Registered as the text/calendar handler by the desktop entry.",
	PreRunE: shellApp.ResolveConfig,
	Args:    cobra.ArbitraryArgs,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "" {
			return shellApp.CallOrLaunch("ui.show", nil)
		}
		for _, arg := range args {
			method, params, err := openCalendarParams(arg)
			if err != nil {
				return err
			}
			if err := shellApp.CallOrLaunch(method, params); err != nil {
				return err
			}
		}
		return nil
	},
}

func viewParams() map[string]any {
	if windowView == "" {
		return nil
	}
	return map[string]any{"view": windowView}
}
