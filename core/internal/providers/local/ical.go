package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

func (p *Provider) readEvents(cal calendar.Calendar) ([]calendar.Event, error) {
	source, err := p.calendarPath(cal)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", source, err)
	}

	switch {
	case info.IsDir():
		return p.readDirectory(cal, source)
	default:
		return p.readFile(cal, source)
	}
}

func (p *Provider) calendarPath(cal calendar.Calendar) (string, error) {
	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		return filepath.Join(p.root, strings.TrimPrefix(cal.RemoteID, "dir:")), nil
	case strings.HasPrefix(cal.RemoteID, "file:"):
		return filepath.Join(p.root, strings.TrimPrefix(cal.RemoteID, "file:")), nil
	}
	return "", fmt.Errorf("unknown remote id %q", cal.RemoteID)
}

func (p *Provider) readDirectory(cal calendar.Calendar, dir string) ([]calendar.Event, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var events []calendar.Event
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".ics") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		evs, err := p.readFile(cal, path)
		if err != nil {
			return nil, err
		}
		events = append(events, evs...)
	}
	return events, nil
}

func (p *Provider) readFile(cal calendar.Calendar, path string) ([]calendar.Event, error) {
	doc, err := loadCalendarDoc(path)
	if err != nil {
		return nil, err
	}

	tz := icalconv.NewTZResolver(doc, cal.TimeZone)
	var events []calendar.Event
	for _, comp := range doc.Events() {
		ev, ok := icalconv.EventFromComponent(cal.ID, comp.Component, tz)
		if !ok {
			continue
		}
		ev.RemoteID = icalconv.ComponentUID(comp.Component)
		events = append(events, ev)
	}
	return events, nil
}
