package local_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/local"
)

func TestSeedDefaultCalendarSeedsEmptyDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "calendars")

	require.NoError(t, local.SeedDefaultCalendar(root, ""))

	require.FileExists(t, filepath.Join(root, "Personal.ics"))

	cals := listCalendars(t, root)
	require.Len(t, cals, 1)
	require.Equal(t, "Personal", cals[0].Name)
}

func TestSeedDefaultCalendarSkipsPopulatedDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "Existing.ics"), []byte(multiEventICS), 0o644))

	require.NoError(t, local.SeedDefaultCalendar(root, ""))

	require.NoFileExists(t, filepath.Join(root, "Personal.ics"))
}

func TestSeedDefaultCalendarIgnoresNonCalendarDirs(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".config"), 0o755))

	require.NoError(t, local.SeedDefaultCalendar(root, ""))

	require.FileExists(t, filepath.Join(root, "Personal.ics"))
}

func TestCreateCalendar(t *testing.T) {
	root := t.TempDir()

	cal, err := local.CreateCalendar(root, "Work Trips")
	require.NoError(t, err)
	require.Equal(t, "Work Trips", cal.Name)
	require.FileExists(t, filepath.Join(root, "Work Trips.ics"))

	cals := listCalendars(t, root)
	require.Len(t, cals, 1)
	require.False(t, cals[0].ReadOnly)
}

func TestCreateCalendarRejectsDuplicate(t *testing.T) {
	root := t.TempDir()

	_, err := local.CreateCalendar(root, "Home")
	require.NoError(t, err)

	_, err = local.CreateCalendar(root, "Home")
	require.Error(t, err)
}

func TestCreateCalendarRejectsBlankName(t *testing.T) {
	_, err := local.CreateCalendar(t.TempDir(), "   ")
	require.Error(t, err)
}

func listCalendars(t *testing.T, root string) []calendar.Calendar {
	t.Helper()
	provider, err := local.New(calendar.Account{ID: "local:test"}, root)
	require.NoError(t, err)
	defer provider.Close()
	cals, err := provider.ListCalendars(context.Background())
	require.NoError(t, err)
	return cals
}
