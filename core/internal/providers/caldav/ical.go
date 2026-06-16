package caldav

import (
	"github.com/emersion/go-webdav/caldav"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

func eventsFromObject(c cal.Calendar, obj caldav.CalendarObject) []cal.Event {
	if obj.Data == nil {
		return nil
	}

	var out []cal.Event
	for _, comp := range obj.Data.Events() {
		ev, ok := icalconv.EventFromComponent(c.ID, comp.Component)
		if !ok {
			continue
		}
		ev.RemoteID = obj.Path
		ev.Etag = obj.ETag
		out = append(out, ev)
	}
	return out
}
