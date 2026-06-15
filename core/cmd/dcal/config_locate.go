package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	configStateName  = "dankcal.path"
	pidFilePrefix    = "dankcal-"
	pidFileExtension = ".pid"
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

func findConfig(_ *cobra.Command, _ []string) error {
	if customConfigPath != "" {
		shellPath := filepath.Join(customConfigPath, "shell.qml")
		info, err := os.Stat(shellPath)
		if err != nil {
			return fmt.Errorf("custom config path: %w", err)
		}
		if info.IsDir() {
			return fmt.Errorf("expected file, got directory: %s", shellPath)
		}
		configPath = customConfigPath
		return nil
	}

	if data, err := os.ReadFile(activeConfigStateFile()); err == nil {
		switch len(getAllPIDs()) {
		case 0:
			os.Remove(activeConfigStateFile())
		default:
			statePath := strings.TrimSpace(string(data))
			shellPath := filepath.Join(statePath, "shell.qml")
			if info, statErr := os.Stat(shellPath); statErr == nil && !info.IsDir() {
				configPath = statePath
				return nil
			}
			os.Remove(activeConfigStateFile())
		}
	}

	return locateDefaultConfig()
}

func locateDefaultConfig() error {
	var primary []string

	if home, err := os.UserConfigDir(); err == nil && home != "" {
		primary = append(primary, filepath.Join(home, "quickshell", "dankcal"))
	}

	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range strings.Split(dataDirs, ":") {
		if dir != "" {
			primary = append(primary, filepath.Join(dir, "quickshell", "dankcal"))
		}
	}

	configDirs := os.Getenv("XDG_CONFIG_DIRS")
	if configDirs == "" {
		configDirs = "/etc/xdg"
	}
	for _, dir := range strings.Split(configDirs, ":") {
		if dir != "" {
			primary = append(primary, filepath.Join(dir, "quickshell", "dankcal"))
		}
	}

	var search []string
	for _, p := range primary {
		search = append(search, p, filepath.Join(p, "quickshell"))
	}

	for _, p := range search {
		shellPath := filepath.Join(p, "shell.qml")
		if info, err := os.Stat(shellPath); err == nil && !info.IsDir() {
			configPath = p
			return nil
		}
	}

	return fmt.Errorf("could not find dankcalendar shell.qml in any XDG location; pass -c <dir>")
}
