package icsimport

import (
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

// Providers have different exception-write APIs. A private import preserves
// the schedule portably: exclude the original occurrence and add its replacement
// as a standalone event. Cancelled exceptions only contribute an exclusion.
func detachExceptions(events []calendar.Event) []calendar.Event {
	masters := make(map[string]int)
	for i, ev := range events {
		if ev.RecurringID == "" {
			masters[ev.UID] = i
		}
	}
	for _, ev := range events {
		if ev.RecurringID == "" {
			continue
		}
		i, found := masters[ev.RecurringID]
		if !found || events[i].Recurrence == nil {
			continue
		}
		format := "20060102T150405Z"
		if events[i].AllDay {
			format = "20060102"
		}
		events[i].Recurrence.ExDate = append(events[i].Recurrence.ExDate, ev.OriginalStart.UTC().Format(format))
	}
	out := events[:0]
	for _, ev := range events {
		if ev.RecurringID != "" {
			if ev.Status == calendar.EventCancelled {
				continue
			}
			ev.RecurringID = ""
			ev.OriginalStart = time.Time{}
			ev.Recurrence = nil
		}
		out = append(out, ev)
	}
	return out
}
