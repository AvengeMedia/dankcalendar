package caldav

import (
	ical "github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

func eventsFromObject(c cal.Calendar, obj caldav.CalendarObject) []cal.Event {
	if obj.Data == nil {
		return nil
	}

	tz := icalconv.NewTZResolver(obj.Data, c.TimeZone)
	var out []cal.Event
	for _, comp := range obj.Data.Events() {
		ev, ok := icalconv.EventFromComponent(c.ID, comp.Component, tz)
		if !ok {
			continue
		}
		ev.RemoteID = obj.Path
		ev.Etag = obj.ETag
		out = append(out, ev)
	}
	return out
}

func tasksFromObject(c cal.Calendar, obj caldav.CalendarObject) []cal.Task {
	if obj.Data == nil {
		return nil
	}

	tz := icalconv.NewTZResolver(obj.Data, c.TimeZone)
	var out []cal.Task
	for _, comp := range obj.Data.Children {
		if comp.Name != ical.CompToDo {
			continue
		}
		t, ok := icalconv.TaskFromComponent(c.ID, comp, tz)
		if !ok {
			continue
		}
		t.RemoteID = obj.Path
		t.Etag = obj.ETag
		out = append(out, t)
	}
	return out
}
