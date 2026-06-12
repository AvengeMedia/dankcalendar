package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const appName = "dankcal"

func ConfigDir() (string, error) {
	dir := filepath.Join(xdg.ConfigHome, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

func DataDir() (string, error) {
	dir := filepath.Join(xdg.DataHome, appName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	return dir, nil
}

func CacheDir() (string, error) {
	dir := filepath.Join(xdg.CacheHome, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}
	return dir, nil
}

func StateDir() (string, error) {
	dir := filepath.Join(xdg.StateHome, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create state dir: %w", err)
	}
	return dir, nil
}

func DatabasePath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dankcal.db"), nil
}

func SocketDir() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return runtimeDir
	}
	return os.TempDir()
}

func SocketPath() string {
	return filepath.Join(SocketDir(), fmt.Sprintf("dankcal-%d.sock", os.Getpid()))
}
