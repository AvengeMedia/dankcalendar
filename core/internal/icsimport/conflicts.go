package icsimport

import (
	"context"
	"strings"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/recurrence"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

// Recurring imports preview a year from the first occurrence; an infinite
// schedule cannot be checked exhaustively. The UI displays this horizon.
func PreviewEnd(ev calendar.Event) time.Time {
	if ev.Recurrence != nil {
		return ev.Start.AddDate(1, 0, 0)
	}
	return ev.End
}

func Conflicts(ctx context.Context, r *repo.Repo, ev calendar.Event) ([]*ent.Event, error) {
	out := make([]*ent.Event, 0)
	if ev.Status == calendar.EventCancelled || strings.EqualFold(ev.Transparency, "TRANSPARENT") {
		return out, nil
	}
	end := PreviewEnd(ev)
	starts := []time.Time{ev.Start}
	if ev.Recurrence != nil {
		rec := ev.Recurrence
		var err error
		starts, err = recurrence.Expand(recurrence.Series{Start: ev.Start, AllDay: ev.AllDay, TimeZone: ev.StartTimeZone, RRule: rec.RRule, RDate: rec.RDate, ExDate: rec.ExDate}, ev.Start, end)
		if err != nil {
			return nil, err
		}
	}
	// All-day dates are stored at UTC midnight but displayed as local dates.
	from, to := ev.Start.Add(-24*time.Hour), end.Add(24*time.Hour)
	candidates, _, err := r.ListEvents(ctx, repo.ListEventsParams{Filter: repo.EventFilter{From: &from, To: &to, IncludeRecurring: true}})
	if err != nil {
		return nil, err
	}
	duration := ev.End.Sub(ev.Start)
	for _, other := range candidates {
		if other.UID == ev.UID || other.RecurringID == ev.UID || string(other.Status) == "cancelled" || strings.EqualFold(other.Transparency, "TRANSPARENT") {
			continue
		}
		otherStart, otherEnd := busyTimes(other.Start, other.End, other.AllDay, time.Local)
		for _, start := range starts {
			ownStart, ownEnd := busyTimes(start, start.Add(duration), ev.AllDay, time.Local)
			if ownStart.Before(otherEnd) && otherStart.Before(ownEnd) {
				out = append(out, other)
				break
			}
		}
	}
	return out, nil
}

func busyTimes(start, end time.Time, allDay bool, loc *time.Location) (time.Time, time.Time) {
	if !allDay {
		return start, end
	}
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc), time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
}
