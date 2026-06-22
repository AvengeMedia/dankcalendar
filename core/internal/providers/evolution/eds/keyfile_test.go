package eds

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKeyFile(t *testing.T) {
	data := "[Data Source]\n" +
		"DisplayName=Holidays in United States\n" +
		"DisplayName[de]=Feiertage\n" +
		"Enabled=true\n" +
		"\n" +
		"# a comment\n" +
		"[Calendar]\n" +
		"BackendName=caldav\n" +
		"Color=#42d692\n" +
		"Selected=true\n"

	kf := parseKeyFile(data)

	require.True(t, kf.hasSection("Calendar"))
	require.False(t, kf.hasSection("Task List"))
	require.Equal(t, "Holidays in United States", kf.get("Data Source", "DisplayName"))
	require.Equal(t, "caldav", kf.get("Calendar", "BackendName"))
	require.Equal(t, "#42d692", kf.get("Calendar", "Color"))
	require.Equal(t, "", kf.get("Calendar", "Missing"))
	require.Equal(t, "", kf.get("Missing", "Key"))
}

func TestHighestVersioned(t *testing.T) {
	names := []string{
		"org.gnome.evolution.dataserver.Calendar7",
		"org.gnome.evolution.dataserver.Calendar8",
		"org.gnome.evolution.dataserver.Sources5",
		"org.freedesktop.DBus",
	}

	require.Equal(t, "org.gnome.evolution.dataserver.Calendar8", highestVersioned(names, calendarPrefix))
	require.Equal(t, "org.gnome.evolution.dataserver.Sources5", highestVersioned(names, sourcesPrefix))
	require.Equal(t, "", highestVersioned([]string{"org.other.Thing"}, calendarPrefix))
}
