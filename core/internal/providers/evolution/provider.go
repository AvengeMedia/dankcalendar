// Package evolution is a calendar provider backed by Evolution Data Server. It
// reads and writes the calendars EDS already manages (local, CalDAV, Google,
// etc.) over D-Bus, so EDS owns credentials and remote sync while dankcalendar
// works against its cache.
package evolution

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ical "github.com/emersion/go-ical"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/evolution/eds"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

type Provider struct {
	account calendar.Account
	client  *eds.Client

	mu   sync.Mutex
	open map[string]*eds.Calendar
}

func New(account calendar.Account) (*Provider, error) {
	client, err := eds.Dial()
	if err != nil {
		return nil, err
	}
	return &Provider{account: account, client: client, open: map[string]*eds.Calendar{}}, nil
}

func (p *Provider) Kind() calendar.AccountKind { return calendar.AccountEvolution }
func (p *Provider) Account() calendar.Account  { return p.account }

func (p *Provider) ListCalendars(ctx context.Context) ([]calendar.Calendar, error) {
	sources, err := p.client.ListCalendarSources()
	if err != nil {
		return nil, err
	}

	out := make([]calendar.Calendar, 0, len(sources))
	for _, s := range sources {
		cal := calendar.Calendar{
			AccountID: p.account.ID,
			RemoteID:  s.UID,
			Name:      s.DisplayName,
			Color:     s.Color,
			ReadOnly:  true,
		}
		if cal.Name == "" {
			cal.Name = s.UID
		}

		switch c, err := p.calendar(s.UID); {
		case err != nil:
			log.Debugf("evolution: open calendar %q: %v", s.UID, err)
		default:
			if writable, err := c.Writable(); err == nil {
				cal.ReadOnly = !writable
			}
		}
		out = append(out, cal)
	}
	return out, nil
}

func (p *Provider) Sync(ctx context.Context, cal calendar.Calendar, cursor calendar.SyncCursor) (*calendar.SyncResult, error) {
	c, err := p.calendar(cal.RemoteID)
	if err != nil {
		return nil, err
	}

	rev, err := c.Revision()
	if err == nil && rev != "" && rev == cursor.Token {
		return &calendar.SyncResult{Cursor: calendar.SyncCursor{CalendarID: cal.ID, Token: rev}}, nil
	}

	objects, err := c.ObjectList("#t")
	if err != nil {
		return nil, err
	}

	events := eventsFromICS(cal.ID, objects)
	changes := make([]calendar.EventChange, 0, len(events))
	for i := range events {
		changes = append(changes, calendar.EventChange{Type: calendar.ChangeUpsert, Event: &events[i]})
	}

	return &calendar.SyncResult{
		Cursor:       calendar.SyncCursor{CalendarID: cal.ID, Token: rev},
		Changes:      changes,
		FullSnapshot: true,
	}, nil
}

func (p *Provider) ListEvents(ctx context.Context, cal calendar.Calendar, opts calendar.ListEventsOptions) ([]calendar.Event, error) {
	c, err := p.calendar(cal.RemoteID)
	if err != nil {
		return nil, err
	}

	objects, err := c.ObjectList(rangeQuery(opts.Start, opts.End))
	if err != nil {
		return nil, err
	}
	return eventsFromICS(cal.ID, objects), nil
}

func (p *Provider) Close() error {
	p.mu.Lock()
	for _, c := range p.open {
		c.Close()
	}
	p.open = map[string]*eds.Calendar{}
	p.mu.Unlock()
	return p.client.Close()
}

// calendar opens a backend once and reuses the handle for the provider's
// lifetime so a sync run does not re-open the same calendar repeatedly.
func (p *Provider) calendar(uid string) (*eds.Calendar, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.open[uid]; ok {
		return c, nil
	}
	c, err := p.client.OpenCalendar(uid)
	if err != nil {
		return nil, err
	}
	p.open[uid] = c
	return c, nil
}

// rangeQuery builds the EDS S-expression for a time window, falling back to
// every object when no bounds are given.
func rangeQuery(start, end time.Time) string {
	if start.IsZero() && end.IsZero() {
		return "#t"
	}
	if start.IsZero() {
		start = time.Unix(0, 0)
	}
	if end.IsZero() {
		end = time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	const layout = "20060102T150405Z"
	return fmt.Sprintf(`(occur-in-time-range? (make-time "%s") (make-time "%s"))`,
		start.UTC().Format(layout), end.UTC().Format(layout))
}

func eventsFromICS(calID string, objects []string) []calendar.Event {
	var events []calendar.Event
	for _, obj := range objects {
		doc, err := ical.NewDecoder(strings.NewReader(wrapComponent(obj))).Decode()
		if err != nil {
			log.Debugf("evolution: decode object: %v", err)
			continue
		}

		tz := icalconv.NewTZResolver(doc, "")
		for _, comp := range doc.Events() {
			ev, ok := icalconv.EventFromComponent(calID, comp.Component, tz)
			if !ok {
				continue
			}
			ev.RemoteID = icalconv.ComponentUID(comp.Component)
			ev.RawICS = obj
			events = append(events, ev)
		}
	}
	return events
}

// wrapComponent wraps the bare VEVENT that GetObjectList returns in a VCALENDAR
// so the iCalendar decoder accepts it.
func wrapComponent(obj string) string {
	if strings.Contains(obj, "BEGIN:VCALENDAR") {
		return obj
	}
	var buf bytes.Buffer
	buf.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//dankcalendar//dankcalendar//EN\r\n")
	buf.WriteString(strings.TrimSpace(obj))
	buf.WriteString("\r\nEND:VCALENDAR\r\n")
	return buf.String()
}
