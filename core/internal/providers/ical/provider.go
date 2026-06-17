package ical

import (
	"context"
	"errors"
	"net/url"

	ical "github.com/emersion/go-ical"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

const remoteID = "feed"

var errReadOnly = errors.New("ical subscriptions are read-only")

type Provider struct {
	account  calendar.Account
	url      string
	username string
	password []byte
	name     string
}

func New(account calendar.Account, feedURL, username string, password []byte, name string) *Provider {
	return &Provider{account: account, url: feedURL, username: username, password: password, name: name}
}

func (p *Provider) Kind() calendar.AccountKind { return calendar.AccountICal }
func (p *Provider) Account() calendar.Account  { return p.account }

func (p *Provider) ListCalendars(ctx context.Context) ([]calendar.Calendar, error) {
	return []calendar.Calendar{{
		AccountID: p.account.ID,
		RemoteID:  remoteID,
		Name:      p.calendarName(),
		ReadOnly:  true,
	}}, nil
}

func (p *Provider) calendarName() string {
	switch {
	case p.name != "":
		return p.name
	case p.account.DisplayName != "":
		return p.account.DisplayName
	}
	if u, err := url.Parse(p.url); err == nil && u.Host != "" {
		return u.Host
	}
	return "Subscription"
}

func (p *Provider) Sync(ctx context.Context, cal calendar.Calendar, cursor calendar.SyncCursor) (*calendar.SyncResult, error) {
	prev := decodeCursor(cursor.Token)
	f, err := fetch(ctx, p.url, p.username, p.password, prev)
	if err != nil {
		return nil, err
	}

	if f.notModified {
		return &calendar.SyncResult{
			Cursor:     calendar.SyncCursor{CalendarID: cal.ID, Token: prev.encode()},
			RetryAfter: prev.ttl(),
		}, nil
	}

	events := eventsFromDoc(cal.ID, f.doc)
	changes := make([]calendar.EventChange, 0, len(events))
	for i := range events {
		changes = append(changes, calendar.EventChange{Type: calendar.ChangeUpsert, Event: &events[i]})
	}

	return &calendar.SyncResult{
		Cursor:       calendar.SyncCursor{CalendarID: cal.ID, Token: f.cursor.encode()},
		Changes:      changes,
		FullSnapshot: true,
		RetryAfter:   f.cursor.ttl(),
	}, nil
}

func (p *Provider) ListEvents(ctx context.Context, cal calendar.Calendar, opts calendar.ListEventsOptions) ([]calendar.Event, error) {
	f, err := fetch(ctx, p.url, p.username, p.password, cursor{})
	if err != nil {
		return nil, err
	}

	events := eventsFromDoc(cal.ID, f.doc)
	if opts.Start.IsZero() && opts.End.IsZero() {
		return events, nil
	}

	out := make([]calendar.Event, 0, len(events))
	for _, ev := range events {
		switch {
		case !opts.Start.IsZero() && ev.End.Before(opts.Start):
			continue
		case !opts.End.IsZero() && ev.Start.After(opts.End):
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

func (p *Provider) CreateEvent(context.Context, calendar.Calendar, *calendar.Event) (*calendar.Event, error) {
	return nil, errReadOnly
}

func (p *Provider) UpdateEvent(context.Context, calendar.Calendar, *calendar.Event) (*calendar.Event, error) {
	return nil, errReadOnly
}

func (p *Provider) DeleteEvent(context.Context, calendar.Calendar, calendar.Event) error {
	return errReadOnly
}

func (p *Provider) Close() error { return nil }

// Probe validates that a feed is reachable and returns the normalized URL plus
// the feed's advertised name (X-WR-CALNAME), used when adding a subscription.
func Probe(ctx context.Context, rawURL, username, password string) (feedURL, name string, err error) {
	feedURL, err = normalizeURL(rawURL)
	if err != nil {
		return "", "", err
	}
	f, err := fetch(ctx, feedURL, username, []byte(password), cursor{})
	if err != nil {
		return "", "", err
	}
	return feedURL, f.name, nil
}

func eventsFromDoc(calID string, doc *ical.Calendar) []calendar.Event {
	if doc == nil {
		return nil
	}

	var events []calendar.Event
	for _, comp := range doc.Events() {
		ev, ok := icalconv.EventFromComponent(calID, comp.Component)
		if !ok {
			continue
		}
		ev.RemoteID = icalconv.ComponentUID(comp.Component)
		events = append(events, ev)
	}
	return events
}
