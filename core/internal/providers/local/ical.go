package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/recurrence"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := ics.ParseCalendar(f)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	var events []calendar.Event
	for _, ev := range doc.Events() {
		converted, err := convertEvent(cal, ev)
		if err != nil {
			return nil, fmt.Errorf("convert event in %q: %w", path, err)
		}
		events = append(events, converted)
	}
	return events, nil
}

func convertEvent(cal calendar.Calendar, ev *ics.VEvent) (calendar.Event, error) {
	uid := ev.GetProperty(ics.ComponentPropertyUniqueId)
	if uid == nil {
		return calendar.Event{}, fmt.Errorf("event missing UID")
	}

	start, err := ev.GetStartAt()
	if err != nil {
		return calendar.Event{}, fmt.Errorf("event start: %w", err)
	}

	end, err := ev.GetEndAt()
	if err != nil {
		end = start
	}

	out := calendar.Event{
		CalendarID:  cal.ID,
		UID:         uid.Value,
		RemoteID:    uid.Value,
		Summary:     propValue(ev, ics.ComponentPropertySummary),
		Description: propValue(ev, ics.ComponentPropertyDescription),
		Location:    propValue(ev, ics.ComponentPropertyLocation),
		URL:         propValue(ev, ics.ComponentPropertyUrl),
		Status:      eventStatus(propValue(ev, ics.ComponentPropertyStatus)),
		Start:       start,
		End:         end,
		AllDay:      isAllDay(ev),
	}

	if rid := ev.GetProperty(ics.ComponentPropertyRecurrenceId); rid != nil {
		out.UID = uid.Value + "/" + rid.Value
		out.RecurringID = uid.Value
		if t, err := recurrence.ParseDateTime(rid.Value, time.UTC); err == nil {
			out.OriginalStart = t
		}
	}

	applyLocalRecurrence(ev, &out)
	applyAlarms(ev, &out)
	return out, nil
}

func applyLocalRecurrence(ev *ics.VEvent, out *calendar.Event) {
	rrules := propValues(ev, ics.ComponentPropertyRrule)
	rdates := propValues(ev, ics.ComponentPropertyRdate)
	exdates := propValues(ev, ics.ComponentPropertyExdate)
	if len(rrules)+len(rdates)+len(exdates) == 0 {
		return
	}
	out.Recurrence = &calendar.Recurrence{RRule: rrules, RDate: rdates, ExDate: exdates}
}

func propValue(ev *ics.VEvent, key ics.ComponentProperty) string {
	prop := ev.GetProperty(key)
	if prop == nil {
		return ""
	}
	return prop.Value
}

func propValues(ev *ics.VEvent, key ics.ComponentProperty) []string {
	var out []string
	for i := range ev.Properties {
		if ics.ComponentProperty(ev.Properties[i].IANAToken) == key && ev.Properties[i].Value != "" {
			out = append(out, ev.Properties[i].Value)
		}
	}
	return out
}

func isAllDay(ev *ics.VEvent) bool {
	prop := ev.GetProperty(ics.ComponentPropertyDtStart)
	if prop == nil {
		return false
	}
	return prop.GetValueType() == ics.ValueDataTypeDate
}

func eventStatus(value string) calendar.EventStatus {
	switch strings.ToUpper(value) {
	case string(ics.ObjectStatusConfirmed):
		return calendar.EventConfirmed
	case string(ics.ObjectStatusTentative):
		return calendar.EventTentative
	case string(ics.ObjectStatusCancelled):
		return calendar.EventCancelled
	}
	return ""
}
