package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

var filenameSanitizer = strings.NewReplacer(
	"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
	"\"", "_", "<", "_", ">", "_", "|", "_", "\x00", "_",
)

func (p *Provider) CreateEvent(ctx context.Context, cal calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	source, err := p.calendarPath(cal)
	if err != nil {
		return nil, err
	}

	uid := ev.UID
	if uid == "" {
		uid = uuid.NewString()
	}

	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		err = createInDirectory(source, uid, ev)
	default:
		err = createInFile(source, uid, ev)
	}
	if err != nil {
		return nil, err
	}
	return storedEvent(cal, uid, ev), nil
}

func (p *Provider) UpdateEvent(ctx context.Context, cal calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	if ev.UID == "" {
		return nil, fmt.Errorf("event missing UID")
	}
	source, err := p.calendarPath(cal)
	if err != nil {
		return nil, err
	}

	path := source
	var doc *ics.Calendar
	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		path, doc, err = findEventFile(source, ev.UID)
	default:
		doc, err = loadCalendarDoc(source)
	}
	if err != nil {
		return nil, err
	}

	if !replaceVEvent(doc, ev.UID, ev) {
		return nil, fmt.Errorf("event not found")
	}
	if err := writeAtomic(path, doc); err != nil {
		return nil, err
	}
	return storedEvent(cal, ev.UID, ev), nil
}

func (p *Provider) DeleteEvent(ctx context.Context, cal calendar.Calendar, ev calendar.Event) error {
	if ev.UID == "" {
		return fmt.Errorf("event missing UID")
	}
	source, err := p.calendarPath(cal)
	if err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		return deleteInDirectory(source, ev.UID)
	default:
		return deleteInFile(source, ev.UID)
	}
}

func createInDirectory(dir, uid string, ev *calendar.Event) error {
	path := filepath.Join(dir, filenameSanitizer.Replace(uid)+".ics")
	switch _, err := os.Stat(path); {
	case err == nil:
		return fmt.Errorf("event %q already exists", uid)
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stat %q: %w", path, err)
	}

	doc := newCalendarDoc()
	doc.AddVEvent(buildVEvent(uid, ev))
	return writeAtomic(path, doc)
}

func createInFile(path, uid string, ev *calendar.Event) error {
	doc, err := loadCalendarDoc(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		doc = newCalendarDoc()
	case err != nil:
		return err
	}

	if findVEvent(doc, uid) != nil {
		return fmt.Errorf("event %q already exists", uid)
	}
	doc.AddVEvent(buildVEvent(uid, ev))
	return writeAtomic(path, doc)
}

func deleteInDirectory(dir, uid string) error {
	path, doc, err := findEventFile(dir, uid)
	if err != nil {
		return err
	}

	removeVEvent(doc, uid)
	if len(doc.Events()) == 0 {
		return os.Remove(path)
	}
	return writeAtomic(path, doc)
}

func deleteInFile(path, uid string) error {
	doc, err := loadCalendarDoc(path)
	if err != nil {
		return err
	}
	if !removeVEvent(doc, uid) {
		return fmt.Errorf("event not found")
	}
	return writeAtomic(path, doc)
}

func findEventFile(dir, uid string) (string, *ics.Calendar, error) {
	fast := filepath.Join(dir, filenameSanitizer.Replace(uid)+".ics")
	if doc, err := loadCalendarDoc(fast); err == nil && findVEvent(doc, uid) != nil {
		return fast, doc, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".ics") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		doc, err := loadCalendarDoc(path)
		if err != nil {
			return "", nil, err
		}
		if findVEvent(doc, uid) != nil {
			return path, doc, nil
		}
	}
	return "", nil, fmt.Errorf("event not found")
}

func loadCalendarDoc(path string) (*ics.Calendar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := ics.ParseCalendar(f)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	return doc, nil
}

func newCalendarDoc() *ics.Calendar {
	doc := ics.NewCalendar()
	doc.SetProductId("-//dankcal//dankcalendar//EN")
	return doc
}

func findVEvent(doc *ics.Calendar, uid string) *ics.VEvent {
	for _, ev := range doc.Events() {
		if ev.Id() == uid {
			return ev
		}
	}
	return nil
}

func replaceVEvent(doc *ics.Calendar, uid string, ev *calendar.Event) bool {
	for i, comp := range doc.Components {
		ve, ok := comp.(*ics.VEvent)
		if !ok || ve.Id() != uid {
			continue
		}
		doc.Components[i] = buildVEvent(uid, ev)
		return true
	}
	return false
}

func removeVEvent(doc *ics.Calendar, uid string) bool {
	for i, comp := range doc.Components {
		ve, ok := comp.(*ics.VEvent)
		if !ok || ve.Id() != uid {
			continue
		}
		doc.Components = append(doc.Components[:i], doc.Components[i+1:]...)
		return true
	}
	return false
}

func buildVEvent(uid string, ev *calendar.Event) *ics.VEvent {
	out := ics.NewEvent(uid)
	out.SetDtStampTime(time.Now().UTC())
	out.SetSummary(ev.Summary)

	switch {
	case ev.AllDay:
		out.SetAllDayStartAt(ev.Start)
		out.SetAllDayEndAt(ev.End)
	default:
		out.SetStartAt(ev.Start)
		out.SetEndAt(ev.End)
	}

	if ev.Description != "" {
		out.SetDescription(ev.Description)
	}
	if ev.Location != "" {
		out.SetLocation(ev.Location)
	}
	if ev.URL != "" {
		out.SetURL(ev.URL)
	}
	if status, ok := icalStatus(ev.Status); ok {
		out.SetStatus(status)
	}
	if ev.Recurrence != nil {
		for _, rule := range ev.Recurrence.RRule {
			out.AddRrule(rule)
		}
	}
	return out
}

func icalStatus(status calendar.EventStatus) (ics.ObjectStatus, bool) {
	switch status {
	case calendar.EventConfirmed:
		return ics.ObjectStatusConfirmed, true
	case calendar.EventTentative:
		return ics.ObjectStatusTentative, true
	case calendar.EventCancelled:
		return ics.ObjectStatusCancelled, true
	}
	return "", false
}

func storedEvent(cal calendar.Calendar, uid string, ev *calendar.Event) *calendar.Event {
	out := *ev
	out.CalendarID = cal.ID
	out.UID = uid
	out.RemoteID = uid
	return &out
}

func writeAtomic(path string, doc *ics.Calendar) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".dankcal-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := doc.SerializeTo(tmp); err != nil {
		tmp.Close()
		return fmt.Errorf("serialize %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
