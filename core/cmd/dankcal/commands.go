package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "dankcal",
	Short:   "Dank Calendar CLI",
	Long:    "Dank Calendar — local, Google, Microsoft, CalDAV, and iCloud calendars in one standalone app.",
	Args:    cobra.NoArgs,
	PreRunE: findConfig,
	RunE: func(_ *cobra.Command, _ []string) error {
		return showOrLaunch("show")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(_ *cobra.Command, _ []string) error {
		if jsonOutput {
			return printJSON(map[string]string{
				"version":   Version,
				"commit":    Commit,
				"buildTime": BuildTime,
			})
		}
		fmt.Printf("dankcal %s (commit %s, built %s)\n", Version, Commit, BuildTime)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&customConfigPath, "config", "c", "", "Path to the dankcal UI config dir (containing shell.qml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output JSON for programmatic usage (where supported)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(restartDetachedCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(ipcCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(syncCmd)
}
