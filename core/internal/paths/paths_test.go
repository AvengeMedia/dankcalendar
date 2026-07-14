package paths_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
)

func setXDGHome(t *testing.T, root string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
}

func TestDirsCreatedUnderXDGHome(t *testing.T) {
	root := t.TempDir()
	setXDGHome(t, root)

	tests := []struct {
		name string
		fn   func() (string, error)
		base string
	}{
		{"config", paths.ConfigDir, "config"},
		{"data", paths.DataDir, "data"},
		{"cache", paths.CacheDir, "cache"},
		{"state", paths.StateDir, "state"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := tc.fn()
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(root, tc.base, "dankcal"), dir)

			info, err := os.Stat(dir)
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestDataDirIsPrivate(t *testing.T) {
	setXDGHome(t, t.TempDir())

	dir, err := paths.DataDir()
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestDatabasePath(t *testing.T) {
	root := t.TempDir()
	setXDGHome(t, root)

	path, err := paths.DatabasePath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "data", "dankcal", "dankcal.db"), path)
}

func TestSocketDir(t *testing.T) {
	runtime := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	assert.Equal(t, runtime, paths.SocketDir())

	t.Setenv("XDG_RUNTIME_DIR", "")
	os.Unsetenv("XDG_RUNTIME_DIR")
	assert.Equal(t, os.TempDir(), paths.SocketDir())
}

func TestSocketPathIsPerProcess(t *testing.T) {
	runtime := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtime)

	path := paths.SocketPath()
	assert.True(t, strings.HasPrefix(path, runtime))
	assert.Contains(t, path, fmt.Sprintf("dankcal-%d.sock", os.Getpid()))
}
