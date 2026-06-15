package main

import (
	"os"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Launch dcal (backend + calendar UI)",
	Long:    "Start the dcal backend (IPC + HTTP) and launch the calendar UI.",
	PreRunE: findConfig,
	RunE: func(cmd *cobra.Command, _ []string) error {
		daemon, _ := cmd.Flags().GetBool("daemon")
		session, _ := cmd.Flags().GetBool("session")
		startHidden, _ = cmd.Flags().GetBool("hidden")
		isDaemonChild, _ := cmd.Flags().GetBool("daemon-child")

		if !isDaemonChild && len(getAllPIDs()) > 0 {
			if err := sendUIAction("show"); err == nil {
				log.Info("dcal already running; showing window")
				return nil
			}
		}

		if v, _ := cmd.Flags().GetString("log-level"); v != "" {
			if err := os.Setenv("DANKCAL_LOG_LEVEL", v); err != nil {
				log.Fatalf("Failed to set DANKCAL_LOG_LEVEL: %v", err)
			}
		}
		if v, _ := cmd.Flags().GetString("log-file"); v != "" {
			if err := os.Setenv("DANKCAL_LOG_FILE", v); err != nil {
				log.Fatalf("Failed to set DANKCAL_LOG_FILE: %v", err)
			}
		}
		log.ApplyEnvOverrides()
		if daemon {
			return runShellDaemon(session)
		}
		return runShellInteractive(session)
	},
}

var restartCmd = &cobra.Command{
	Use:     "restart",
	Short:   "Restart the running dcal instance",
	PreRunE: findConfig,
	Run: func(_ *cobra.Command, _ []string) {
		restartShell()
	},
}

var restartDetachedCmd = &cobra.Command{
	Use:    "restart-detached <pid>",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		runDetachedRestart(args[0])
	},
}

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill running dcal processes",
	Run: func(_ *cobra.Command, _ []string) {
		killShell()
	},
}

func init() {
	runCmd.Flags().BoolP("daemon", "d", false, "Run in daemon (background) mode")
	runCmd.Flags().Bool("daemon-child", false, "Internal flag for daemon child process")
	runCmd.Flags().Bool("session", false, "Session managed (e.g. systemd unit)")
	runCmd.Flags().Bool("hidden", false, "Start with the calendar window hidden (tray only)")
	runCmd.Flags().String("log-level", "", "Log level: debug, info, warn, error, fatal (overrides DANKCAL_LOG_LEVEL)")
	runCmd.Flags().String("log-file", "", "Append logs to this file in addition to stderr (overrides DANKCAL_LOG_FILE)")
	_ = runCmd.Flags().MarkHidden("daemon-child")
}
