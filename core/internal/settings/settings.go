// Package settings reads the UI settings file that the quickshell frontend
// owns and writes. The daemon only consumes it, so values are re-read on
// demand rather than cached; defaults here must stay in sync with the
// JsonAdapter defaults in quickshell/Services/SettingsData.qml.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

const fileName = "ui-settings.json"

type UISettings struct {
	// Use24HourClock is the GUI's resolved time format: the QML side folds
	// its tri-state timeFormat ("auto" follows the Qt locale) into this flag
	// because the daemon has no access to Qt locale data.
	Use24HourClock           bool   `json:"use24HourClock"`
	RemindersEnabled         bool   `json:"remindersEnabled"`
	ReminderPersist          bool   `json:"reminderPersist"`
	AllDayReminders          bool   `json:"allDayReminders"`
	AllDayReminderTime       string `json:"allDayReminderTime"`
	AllDayReminderDaysBefore int    `json:"allDayReminderDaysBefore"`
	DefaultReminderMinutes   int    `json:"defaultReminderMinutes"`
	SnoozeMinutes            int    `json:"snoozeMinutes"`
}

func Defaults() UISettings {
	return UISettings{
		Use24HourClock:           true,
		RemindersEnabled:         true,
		ReminderPersist:          true,
		AllDayReminders:          false,
		AllDayReminderTime:       "09:00",
		AllDayReminderDaysBefore: 0,
		DefaultReminderMinutes:   10,
		SnoozeMinutes:            5,
	}
}

func Path() string {
	return filepath.Join(xdg.ConfigHome, "dankcal", fileName)
}

// Load returns defaults when the file is missing or unreadable; a partial
// file only overrides the keys it contains.
func Load() UISettings {
	return loadFrom(Path())
}

func loadFrom(path string) UISettings {
	out := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Defaults()
	}
	if out.SnoozeMinutes <= 0 {
		out.SnoozeMinutes = Defaults().SnoozeMinutes
	}
	return out
}

// AllDayClock parses AllDayReminderTime, falling back to the default on
// malformed input.
func (s UISettings) AllDayClock() (hour, minute int) {
	t, err := time.Parse("15:04", s.AllDayReminderTime)
	if err != nil {
		t, _ = time.Parse("15:04", Defaults().AllDayReminderTime)
	}
	return t.Hour(), t.Minute()
}
