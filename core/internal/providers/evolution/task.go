package evolution

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	ical "github.com/emersion/go-ical"
	"github.com/google/uuid"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/evolution/eds"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

func (p *Provider) CreateTask(ctx context.Context, cal calendar.Calendar, t *calendar.Task) (*calendar.Task, error) {
	c, err := p.writableTaskList(cal.RemoteID)
	if err != nil {
		return nil, err
	}

	uid := t.UID
	if uid == "" {
		uid = uuid.NewString()
	}
	ics, err := encodeTaskICS(t, uid)
	if err != nil {
		return nil, err
	}

	uids, err := c.CreateObjects([]string{ics})
	if err != nil {
		return nil, err
	}
	if len(uids) > 0 && uids[0] != "" {
		uid = uids[0]
	}
	return storedTask(cal, uid, t), nil
}

func (p *Provider) UpdateTask(ctx context.Context, cal calendar.Calendar, t *calendar.Task) (*calendar.Task, error) {
	if t.UID == "" {
		return nil, fmt.Errorf("task missing UID")
	}
	c, err := p.writableTaskList(cal.RemoteID)
	if err != nil {
		return nil, err
	}

	ics, err := encodeTaskICS(t, t.UID)
	if err != nil {
		return nil, err
	}
	if err := c.ModifyObjects([]string{ics}, eds.ModAll); err != nil {
		return nil, err
	}
	return storedTask(cal, t.UID, t), nil
}

func (p *Provider) DeleteTask(ctx context.Context, cal calendar.Calendar, t calendar.Task) error {
	if t.UID == "" {
		return fmt.Errorf("task missing UID")
	}
	c, err := p.writableTaskList(cal.RemoteID)
	if err != nil {
		return err
	}
	return c.RemoveObjects([]eds.UIDRID{{UID: t.UID}}, eds.ModAll)
}

func (p *Provider) writableTaskList(uid string) (*eds.Calendar, error) {
	c, err := p.taskCalendar(uid)
	if err != nil {
		return nil, err
	}
	switch writable, err := c.Writable(); {
	case err != nil:
		return nil, err
	case !writable:
		return nil, errReadOnly
	}
	return c, nil
}

func tasksFromICS(calID string, objects []string) []calendar.Task {
	var tasks []calendar.Task
	for _, obj := range objects {
		doc, err := ical.NewDecoder(strings.NewReader(wrapComponent(obj))).Decode()
		if err != nil {
			continue
		}

		tz := icalconv.NewTZResolver(doc, "")
		for _, comp := range doc.Children {
			if comp.Name != ical.CompToDo {
				continue
			}
			t, ok := icalconv.TaskFromComponent(calID, comp, tz)
			if !ok {
				continue
			}
			t.RemoteID = icalconv.ComponentUID(comp)
			t.RawICS = obj
			tasks = append(tasks, t)
		}
	}
	return tasks
}

// encodeTaskICS returns a bare VTODO. EDS parses each create/modify argument as
// a single component and rejects a VCALENDAR wrapper, mirroring encodeICS.
func encodeTaskICS(t *calendar.Task, uid string) (string, error) {
	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(icalconv.CalendarFromTask(t, uid)); err != nil {
		return "", fmt.Errorf("encode task: %w", err)
	}

	doc := buf.String()
	begin := strings.Index(doc, "BEGIN:VTODO")
	end := strings.LastIndex(doc, "END:VTODO")
	if begin < 0 || end < 0 {
		return "", fmt.Errorf("encode task: no VTODO produced")
	}
	return doc[begin : end+len("END:VTODO")], nil
}

func storedTask(cal calendar.Calendar, uid string, t *calendar.Task) *calendar.Task {
	out := *t
	out.CalendarID = cal.ID
	out.UID = uid
	out.RemoteID = uid
	return &out
}
