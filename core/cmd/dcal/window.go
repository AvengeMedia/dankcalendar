package main

import (
	"path/filepath"

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
	Use:     "open [url|file.ics]",
	Short:   "Open a webcal:// subscription link or an .ics file (no argument just shows the window)",
	Long:    "Open a webcal:// subscription link in the UI, or an .ics file (e.g. an emailed invitation) in the import dialog. Registered as the text/calendar handler by the desktop entry.",
	PreRunE: shellApp.ResolveConfig,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "" {
			return shellApp.CallOrLaunch("ui.show", nil)
		}
		path, isFile := icsFilePath(args[0])
		if !isFile {
			return shellApp.CallOrLaunch("ui.open", map[string]any{"url": args[0]})
		}
		ics, err := readICSFile(path)
		if err != nil {
			return err
		}
		return shellApp.CallOrLaunch("ui.openIcs", map[string]any{"ics": ics, "name": filepath.Base(path)})
	},
}

func viewParams() map[string]any {
	if windowView == "" {
		return nil
	}
	return map[string]any{"view": windowView}
}
