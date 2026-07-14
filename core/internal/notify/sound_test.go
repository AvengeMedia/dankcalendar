package notify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSound(t *testing.T, dataDir, theme, name string) string {
	t.Helper()
	stereo := filepath.Join(dataDir, "sounds", theme, "stereo")
	require.NoError(t, os.MkdirAll(stereo, 0o755))
	file := filepath.Join(stereo, name)
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	return file
}

func TestThemeSoundFile(t *testing.T) {
	dataDir := t.TempDir()
	t.Cleanup(xdg.Reload)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dataDir, "nonexistent"))
	t.Setenv("XDG_DATA_DIRS", dataDir)
	xdg.Reload()

	assert.Empty(t, themeSoundFile("message-new-instant"))

	fallback := writeSound(t, dataDir, "freedesktop", "message-new-instant.oga")
	assert.Equal(t, fallback, themeSoundFile("message-new-instant"))
	alarm := writeSound(t, dataDir, "freedesktop", "alarm-clock-elapsed.oga")
	assert.Equal(t, alarm, themeSoundFile(SoundTask))
	assert.Empty(t, themeSoundFile("no-such-event"))
}

func TestChimeFile(t *testing.T) {
	t.Cleanup(xdg.Reload)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	xdg.Reload()

	file, err := chimeFile()
	require.NoError(t, err)
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, reminderChime, data)

	// A stale or corrupted extraction is rewritten on the next lookup.
	require.NoError(t, os.WriteFile(file, []byte("stale"), 0o644))
	again, err := chimeFile()
	require.NoError(t, err)
	assert.Equal(t, file, again)
	data, err = os.ReadFile(again)
	require.NoError(t, err)
	assert.Equal(t, reminderChime, data)
}
