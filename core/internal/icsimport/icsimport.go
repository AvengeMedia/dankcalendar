// Package icsimport turns iCalendar data from outside the synced providers,
// such as an emailed invitation, into domain events and adds them to a calendar.
package icsimport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	ical "github.com/emersion/go-ical"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

// MaxBytes bounds the data accepted from a file or IPC param; it must stay
// well under the IPC line limit after JSON escaping.
const MaxBytes = 256 << 10

var ErrNoEvents = errors.New("no importable events found")

type Document struct {
	Method string           `json:"method,omitempty"`
	Events []calendar.Event `json:"events"`
}

// Parse decodes every VCALENDAR in data. Recurrence exceptions are skipped:
// they only make sense written alongside their series master.
func Parse(data []byte) (*Document, error) {
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("calendar data exceeds %d KiB", MaxBytes>>10)
	}

	dec := ical.NewDecoder(bytes.NewReader(data))
	doc := &Document{}
	for {
		cal, err := dec.Decode()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse icalendar: %w", err)
		}
		if doc.Method == "" {
			doc.Method = methodOf(cal)
		}
		tz := icalconv.NewTZResolver(cal, "")
		for _, comp := range cal.Events() {
			ev, ok := icalconv.EventFromComponent("", comp.Component, tz)
			if !ok || ev.RecurringID != "" {
				continue
			}
			doc.Events = append(doc.Events, ev)
		}
	}

	if len(doc.Events) == 0 {
		return nil, ErrNoEvents
	}
	return doc, nil
}

func methodOf(cal *ical.Calendar) string {
	prop := cal.Props.Get(ical.PropMethod)
	if prop == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(prop.Value))
}

// Create adds ev to cal through provider, keeping organizer and attendees only
// where the provider can store a foreign event as a private copy.
func Create(ctx context.Context, provider calendar.Provider, cal calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	if importer, ok := provider.(calendar.Importer); ok {
		return importer.ImportEvent(ctx, cal, ev)
	}
	private := *ev
	private.Organizer = nil
	private.Attendees = nil
	return provider.CreateEvent(ctx, cal, &private)
}
