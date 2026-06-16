package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

const defaultCalendarName = "Personal"

// SeedDefaultCalendar writes a starter calendar into root when it holds none,
// so a freshly added local account is immediately writable. It is a no-op when
// the directory already exposes calendars (e.g. an existing vdirsyncer tree).
func SeedDefaultCalendar(root, name string) error {
	abs, err := resolveRoot(root)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return fmt.Errorf("read local root: %w", err)
	}
	for _, e := range entries {
		switch {
		case e.IsDir():
			return nil
		case strings.HasSuffix(strings.ToLower(e.Name()), ".ics"):
			return nil
		}
	}

	if strings.TrimSpace(name) == "" {
		name = defaultCalendarName
	}
	_, err = createCalendarFile(abs, name)
	return err
}

// CreateCalendar adds a new empty calendar (an .ics file) to a local account's
// directory and returns it.
func CreateCalendar(root, name string) (calendar.Calendar, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return calendar.Calendar{}, errors.New("calendar name is required")
	}
	abs, err := resolveRoot(root)
	if err != nil {
		return calendar.Calendar{}, err
	}
	return createCalendarFile(abs, name)
}

func resolveRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve local root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("create local root %q: %w", abs, err)
	}
	return abs, nil
}

func createCalendarFile(dir, name string) (calendar.Calendar, error) {
	file := strings.TrimSpace(filenameSanitizer.Replace(name))
	if file == "" {
		file = defaultCalendarName
	}
	file += ".ics"
	path := filepath.Join(dir, file)

	switch _, err := os.Stat(path); {
	case err == nil:
		return calendar.Calendar{}, fmt.Errorf("a calendar named %q already exists", name)
	case !errors.Is(err, os.ErrNotExist):
		return calendar.Calendar{}, fmt.Errorf("stat %q: %w", path, err)
	}

	if err := writeAtomic(path, icalconv.NewCalendar()); err != nil {
		return calendar.Calendar{}, err
	}
	return calendar.Calendar{
		RemoteID: "file:" + file,
		Name:     strings.TrimSuffix(file, filepath.Ext(file)),
	}, nil
}
