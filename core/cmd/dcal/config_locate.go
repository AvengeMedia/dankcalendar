package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AvengeMedia/dankcalendar/core/internal/shellembed"
	"github.com/spf13/cobra"
)

const (
	configStateName  = "dankcal.path"
	pidFilePrefix    = "dankcal-"
	pidFileExtension = ".pid"
	shellDirEnv      = "DANKCAL_SHELL_DIR"
)

var (
	customConfigPath string
	configPath       string
)

func runtimeDir() string {
	if v := os.Getenv("XDG_RUNTIME_DIR"); v != "" {
		return v
	}
	return os.TempDir()
}

func activeConfigStateFile() string {
	return filepath.Join(runtimeDir(), configStateName)
}

// findConfig resolves the quickshell config dir. Explicit overrides win
// (-c flag, then DANKCAL_SHELL_DIR), then the path a running instance is
// using, then the UI embedded in the binary. There is no implicit
// filesystem search.
func findConfig(_ *cobra.Command, _ []string) error {
	if customConfigPath != "" {
		if err := validateShellDir(customConfigPath); err != nil {
			return fmt.Errorf("custom config path: %w", err)
		}
		configPath = customConfigPath
		return nil
	}

	if dir := os.Getenv(shellDirEnv); dir != "" {
		if err := validateShellDir(dir); err != nil {
			return fmt.Errorf("%s: %w", shellDirEnv, err)
		}
		configPath = dir
		return nil
	}

	if data, err := os.ReadFile(activeConfigStateFile()); err == nil {
		switch len(getAllPIDs()) {
		case 0:
			os.Remove(activeConfigStateFile())
		default:
			statePath := strings.TrimSpace(string(data))
			if err := validateShellDir(statePath); err == nil {
				configPath = statePath
				return nil
			}
			os.Remove(activeConfigStateFile())
		}
	}

	return useEmbeddedShell()
}

func validateShellDir(dir string) error {
	shellPath := filepath.Join(dir, "shell.qml")
	info, err := os.Stat(shellPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file, got directory: %s", shellPath)
	}
	return nil
}

// useEmbeddedShell unpacks the embedded UI into the runtime dir:
// read-only and verified against the embedded content on every
// resolution. Local edits never survive; use -c or DANKCAL_SHELL_DIR
// to run a modified UI.
func useEmbeddedShell() error {
	if !shellembed.Available() {
		return fmt.Errorf("this dcal build has no embedded UI; pass -c <dir> or set %s", shellDirEnv)
	}

	baseDir := filepath.Join(runtimeDir(), "dankcal-shell")
	dir, err := shellembed.Extract(baseDir)
	if err != nil {
		return fmt.Errorf("extract embedded UI: %w", err)
	}

	if len(getAllPIDs()) == 0 {
		shellembed.Prune(baseDir, dir)
	}

	configPath = dir
	return nil
}
