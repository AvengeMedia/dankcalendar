package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    func(UISettings) UISettings
	}{
		{
			name:    "missing file returns defaults",
			content: "",
			want:    func(d UISettings) UISettings { return d },
		},
		{
			name:    "invalid json returns defaults",
			content: "{not json",
			want:    func(d UISettings) UISettings { return d },
		},
		{
			name:    "partial file overrides only present keys",
			content: `{"remindersEnabled": false, "snoozeMinutes": 15}`,
			want: func(d UISettings) UISettings {
				d.RemindersEnabled = false
				d.SnoozeMinutes = 15
				return d
			},
		},
		{
			name:    "zero snooze falls back to default",
			content: `{"snoozeMinutes": 0}`,
			want:    func(d UISettings) UISettings { return d },
		},
		{
			name:    "full file",
			content: `{"use24HourClock": false, "remindersEnabled": true, "reminderPersist": false, "allDayReminders": true, "allDayReminderTime": "18:30", "allDayReminderDaysBefore": 1, "defaultReminderMinutes": -1, "snoozeMinutes": 30}`,
			want: func(d UISettings) UISettings {
				return UISettings{
					Use24HourClock:           false,
					RemindersEnabled:         true,
					ReminderPersist:          false,
					AllDayReminders:          true,
					AllDayReminderTime:       "18:30",
					AllDayReminderDaysBefore: 1,
					DefaultReminderMinutes:   -1,
					SnoozeMinutes:            30,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ui-settings.json")
			if tt.content != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))
			}
			assert.Equal(t, tt.want(Defaults()), loadFrom(path))
		})
	}
}

func TestAllDayClock(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		hour   int
		minute int
	}{
		{"valid", "18:30", 18, 30},
		{"midnight", "00:00", 0, 0},
		{"malformed falls back", "9am", 9, 0},
		{"empty falls back", "", 9, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Defaults()
			s.AllDayReminderTime = tt.value
			hour, minute := s.AllDayClock()
			assert.Equal(t, tt.hour, hour)
			assert.Equal(t, tt.minute, minute)
		})
	}
}
