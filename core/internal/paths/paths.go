package paths

import (
	"path/filepath"

	dankpaths "github.com/AvengeMedia/dankgo/paths"
)

var app = dankpaths.New("dankcal")

func ConfigDir() (string, error) { return app.ConfigDir() }

func DataDir() (string, error) { return app.DataDir() }

func CacheDir() (string, error) { return app.CacheDir() }

func StateDir() (string, error) { return app.StateDir() }

func DatabasePath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dankcal.db"), nil
}

func SocketDir() string { return app.SocketDir() }

func SocketPath() string { return app.SocketPath() }

// ShowAfterRestartPath marks a restart requested from the UI, so the next
// launch shows the window even when the service starts with --hidden.
func ShowAfterRestartPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "show-after-restart"), nil
}
