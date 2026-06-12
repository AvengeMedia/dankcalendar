package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

type Provider struct {
	account calendar.Account
	root    string
}

func New(account calendar.Account, root string) (*Provider, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local root: %w", err)
	}
	info, err := os.Stat(abs)
	switch {
	case os.IsNotExist(err):
		if mkErr := os.MkdirAll(abs, 0o755); mkErr != nil {
			return nil, fmt.Errorf("create local root %q: %w", abs, mkErr)
		}
	case err != nil:
		return nil, fmt.Errorf("stat local root: %w", err)
	case !info.IsDir():
		return nil, fmt.Errorf("local root %q is not a directory", abs)
	}
	return &Provider{account: account, root: abs}, nil
}

func (p *Provider) Kind() calendar.AccountKind { return calendar.AccountLocal }
func (p *Provider) Account() calendar.Account  { return p.account }

func (p *Provider) ListCalendars(ctx context.Context) ([]calendar.Calendar, error) {
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return nil, fmt.Errorf("read local root: %w", err)
	}

	var calendars []calendar.Calendar
	for _, entry := range entries {
		switch {
		case entry.IsDir():
			calendars = append(calendars, p.directoryCalendar(entry.Name()))
		case strings.HasSuffix(strings.ToLower(entry.Name()), ".ics"):
			calendars = append(calendars, p.fileCalendar(entry.Name()))
		}
	}
	return calendars, nil
}

func (p *Provider) directoryCalendar(name string) calendar.Calendar {
	return calendar.Calendar{
		AccountID: p.account.ID,
		RemoteID:  "dir:" + name,
		Name:      name,
	}
}

func (p *Provider) fileCalendar(name string) calendar.Calendar {
	return calendar.Calendar{
		AccountID: p.account.ID,
		RemoteID:  "file:" + name,
		Name:      strings.TrimSuffix(name, filepath.Ext(name)),
	}
}

func (p *Provider) Sync(ctx context.Context, cal calendar.Calendar, cursor calendar.SyncCursor) (*calendar.SyncResult, error) {
	events, err := p.readEvents(cal)
	if err != nil {
		return nil, err
	}

	changes := make([]calendar.EventChange, 0, len(events))
	for i := range events {
		ev := events[i]
		changes = append(changes, calendar.EventChange{Type: calendar.ChangeUpsert, Event: &ev})
	}

	return &calendar.SyncResult{
		Cursor:       calendar.SyncCursor{CalendarID: cal.ID},
		Changes:      changes,
		FullSnapshot: true,
	}, nil
}

func (p *Provider) ListEvents(ctx context.Context, cal calendar.Calendar, opts calendar.ListEventsOptions) ([]calendar.Event, error) {
	events, err := p.readEvents(cal)
	if err != nil {
		return nil, err
	}

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

func (p *Provider) Close() error { return nil }
