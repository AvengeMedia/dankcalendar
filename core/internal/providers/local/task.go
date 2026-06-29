package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ical "github.com/emersion/go-ical"
	"github.com/google/uuid"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

func (p *Provider) readTasks(cal calendar.Calendar) ([]calendar.Task, error) {
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
		return p.readTaskDirectory(cal, source)
	default:
		return p.readTaskFile(cal, source)
	}
}

func (p *Provider) readTaskDirectory(cal calendar.Calendar, dir string) ([]calendar.Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var tasks []calendar.Task
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".ics") {
			continue
		}
		ts, err := p.readTaskFile(cal, filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, ts...)
	}
	return tasks, nil
}

func (p *Provider) readTaskFile(cal calendar.Calendar, path string) ([]calendar.Task, error) {
	doc, err := loadCalendarDoc(path)
	if err != nil {
		return nil, err
	}

	tz := icalconv.NewTZResolver(doc, cal.TimeZone)
	var tasks []calendar.Task
	for _, comp := range doc.Children {
		if comp.Name != ical.CompToDo {
			continue
		}
		t, ok := icalconv.TaskFromComponent(cal.ID, comp, tz)
		if !ok {
			continue
		}
		t.RemoteID = icalconv.ComponentUID(comp)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (p *Provider) CreateTask(ctx context.Context, cal calendar.Calendar, t *calendar.Task) (*calendar.Task, error) {
	source, err := p.calendarPath(cal)
	if err != nil {
		return nil, err
	}

	uid := t.UID
	if uid == "" {
		uid = uuid.NewString()
	}

	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		err = createTaskInDirectory(source, uid, t)
	default:
		err = createTaskInFile(source, uid, t)
	}
	if err != nil {
		return nil, err
	}
	return storedTask(cal, uid, t), nil
}

func (p *Provider) UpdateTask(ctx context.Context, cal calendar.Calendar, t *calendar.Task) (*calendar.Task, error) {
	if t.UID == "" {
		return nil, fmt.Errorf("task missing UID")
	}
	source, err := p.calendarPath(cal)
	if err != nil {
		return nil, err
	}

	path := source
	var doc *ical.Calendar
	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		path, doc, err = findComponentFile(source, ical.CompToDo, t.UID)
	default:
		doc, err = loadCalendarDoc(source)
	}
	if err != nil {
		return nil, err
	}

	if !replaceComponent(doc, ical.CompToDo, t.UID, icalconv.BuildTask(t, t.UID)) {
		return nil, fmt.Errorf("task not found")
	}
	if err := writeAtomic(path, doc); err != nil {
		return nil, err
	}
	return storedTask(cal, t.UID, t), nil
}

func (p *Provider) DeleteTask(ctx context.Context, cal calendar.Calendar, t calendar.Task) error {
	if t.UID == "" {
		return fmt.Errorf("task missing UID")
	}
	source, err := p.calendarPath(cal)
	if err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(cal.RemoteID, "dir:"):
		return deleteTaskInDirectory(source, t.UID)
	default:
		return deleteTaskInFile(source, t.UID)
	}
}

func createTaskInDirectory(dir, uid string, t *calendar.Task) error {
	path := filepath.Join(dir, filenameSanitizer.Replace(uid)+".ics")
	switch _, err := os.Stat(path); {
	case err == nil:
		return fmt.Errorf("task %q already exists", uid)
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stat %q: %w", path, err)
	}

	doc := icalconv.NewCalendar()
	doc.Children = append(doc.Children, icalconv.BuildTask(t, uid))
	return writeAtomic(path, doc)
}

func createTaskInFile(path, uid string, t *calendar.Task) error {
	doc, err := loadCalendarDoc(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		doc = icalconv.NewCalendar()
	case err != nil:
		return err
	}

	if findComponent(doc, ical.CompToDo, uid) != nil {
		return fmt.Errorf("task %q already exists", uid)
	}
	doc.Children = append(doc.Children, icalconv.BuildTask(t, uid))
	return writeAtomic(path, doc)
}

func deleteTaskInDirectory(dir, uid string) error {
	path, doc, err := findComponentFile(dir, ical.CompToDo, uid)
	if err != nil {
		return err
	}

	removeComponent(doc, ical.CompToDo, uid)
	if len(doc.Children) == 0 {
		return os.Remove(path)
	}
	return writeAtomic(path, doc)
}

func deleteTaskInFile(path, uid string) error {
	doc, err := loadCalendarDoc(path)
	if err != nil {
		return err
	}
	if !removeComponent(doc, ical.CompToDo, uid) {
		return fmt.Errorf("task not found")
	}
	return writeAtomic(path, doc)
}

func storedTask(cal calendar.Calendar, uid string, t *calendar.Task) *calendar.Task {
	out := *t
	out.CalendarID = cal.ID
	out.UID = uid
	out.RemoteID = uid
	return &out
}
