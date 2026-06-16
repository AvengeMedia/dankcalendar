package caldav

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

type Provider struct {
	account    cal.Account
	client     *caldav.Client
	httpClient webdav.HTTPClient
	endpoint   *url.URL
	homeSet    string
	username   string
}

func New(ctx context.Context, account cal.Account, secrets cal.SecretStore, baseURL, username string) (*Provider, error) {
	password, err := secrets.Get(ctx, account.ID, SecretKeyPassword)
	if err != nil {
		return nil, fmt.Errorf("caldav password not stored for %q: %w", account.ID, err)
	}

	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse caldav url: %w", err)
	}

	httpClient := webdav.HTTPClientWithBasicAuth(http.DefaultClient, username, string(password))
	client, err := caldav.NewClient(httpClient, baseURL)
	if err != nil {
		return nil, fmt.Errorf("init caldav client: %w", err)
	}

	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("find caldav principal: %w", err)
	}
	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("find caldav home set: %w", err)
	}

	return &Provider{
		account:    account,
		client:     client,
		httpClient: httpClient,
		endpoint:   endpoint,
		homeSet:    homeSet,
		username:   username,
	}, nil
}

func (p *Provider) Kind() cal.AccountKind { return cal.AccountCalDAV }
func (p *Provider) Account() cal.Account  { return p.account }

func (p *Provider) ListCalendars(ctx context.Context) ([]cal.Calendar, error) {
	calendars, err := p.client.FindCalendars(ctx, p.homeSet)
	if err != nil {
		return nil, fmt.Errorf("find caldav calendars: %w", err)
	}

	out := make([]cal.Calendar, 0, len(calendars))
	for _, c := range calendars {
		out = append(out, cal.Calendar{
			AccountID:   p.account.ID,
			RemoteID:    c.Path,
			Name:        c.Name,
			Description: c.Description,
		})
	}
	return out, nil
}

func (p *Provider) Sync(ctx context.Context, c cal.Calendar, cursor cal.SyncCursor) (*cal.SyncResult, error) {
	objects, err := p.queryEvents(ctx, c.RemoteID, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("query caldav calendar %q: %w", c.RemoteID, err)
	}

	result := &cal.SyncResult{
		Cursor:       cal.SyncCursor{CalendarID: cursor.CalendarID},
		FullSnapshot: true,
	}
	for _, obj := range objects {
		for _, ev := range eventsFromObject(c, obj) {
			ev := ev
			result.Changes = append(result.Changes, cal.EventChange{Type: cal.ChangeUpsert, Event: &ev})
		}
	}
	return result, nil
}

func (p *Provider) ListEvents(ctx context.Context, c cal.Calendar, opts cal.ListEventsOptions) ([]cal.Event, error) {
	objects, err := p.queryEvents(ctx, c.RemoteID, opts.Start, opts.End)
	if err != nil {
		return nil, fmt.Errorf("query caldav calendar %q: %w", c.RemoteID, err)
	}

	var events []cal.Event
	for _, obj := range objects {
		events = append(events, eventsFromObject(c, obj)...)
	}
	if opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}
	return events, nil
}

func (p *Provider) CreateEvent(ctx context.Context, c cal.Calendar, ev *cal.Event) (*cal.Event, error) {
	uid := ev.UID
	if uid == "" {
		uid = uuid.NewString()
	}

	path := objectPath(c.RemoteID, uid)
	obj, err := p.client.PutCalendarObject(ctx, path, icalconv.CalendarFromEvent(ev, uid))
	if err != nil {
		return nil, fmt.Errorf("put caldav object %q: %w", path, err)
	}

	out := *ev
	out.UID = uid
	out.RemoteID = path
	out.Etag = obj.ETag
	return &out, nil
}

func (p *Provider) UpdateEvent(ctx context.Context, c cal.Calendar, ev *cal.Event) (*cal.Event, error) {
	if ev.RemoteID == "" {
		return nil, errors.New("caldav update requires remote id")
	}

	obj, err := p.client.PutCalendarObject(ctx, ev.RemoteID, icalconv.CalendarFromEvent(ev, ev.UID))
	if err != nil {
		return nil, fmt.Errorf("put caldav object %q: %w", ev.RemoteID, err)
	}

	out := *ev
	out.Etag = obj.ETag
	return &out, nil
}

func (p *Provider) DeleteEvent(ctx context.Context, c cal.Calendar, ev cal.Event) error {
	if ev.RemoteID == "" {
		return errors.New("caldav delete requires remote id")
	}
	if err := p.client.RemoveAll(ctx, ev.RemoteID); err != nil {
		return fmt.Errorf("delete caldav object %q: %w", ev.RemoteID, err)
	}
	return nil
}

func (p *Provider) queryEvents(ctx context.Context, calendarPath string, start, end time.Time) ([]caldav.CalendarObject, error) {
	eventFilter := caldav.CompFilter{Name: ical.CompEvent}
	if !start.IsZero() {
		eventFilter.Start = start
	}
	if !end.IsZero() {
		eventFilter.End = end
	}

	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{Name: ical.CompCalendar, AllProps: true, AllComps: true},
		CompFilter:  caldav.CompFilter{Name: ical.CompCalendar, Comps: []caldav.CompFilter{eventFilter}},
	}

	objects, queryErr := p.client.QueryCalendar(ctx, calendarPath, query)
	if queryErr == nil {
		return objects, nil
	}

	// Some servers (notably iCloud) refuse to return calendar-data from a
	// calendar-query REPORT; list the collection and multiget instead.
	objects, err := p.multiGetAll(ctx, calendarPath)
	if err != nil {
		return nil, errors.Join(queryErr, err)
	}

	filtered, err := caldav.Filter(query, objects)
	if err != nil {
		return objects, nil
	}
	return filtered, nil
}

func (p *Provider) multiGetAll(ctx context.Context, calendarPath string) ([]caldav.CalendarObject, error) {
	paths, err := p.listObjectPaths(ctx, calendarPath)
	if err != nil {
		return nil, fmt.Errorf("list caldav objects: %w", err)
	}
	if len(paths) == 0 {
		return nil, nil
	}

	req := caldav.CalendarCompRequest{Name: ical.CompCalendar, AllProps: true, AllComps: true}
	var out []caldav.CalendarObject
	for chunk := range slices.Chunk(paths, 50) {
		objs, err := p.client.MultiGetCalendar(ctx, calendarPath, &caldav.CalendarMultiGet{
			Paths:       chunk,
			CompRequest: req,
		})
		if err != nil {
			return nil, fmt.Errorf("multiget caldav objects: %w", err)
		}
		out = append(out, objs...)
	}
	return out, nil
}

// Raw PROPFIND asking only for hrefs: go-webdav's higher-level helpers error
// on iCloud's 404 propstats for props we don't even need.
const listObjectsBody = `<?xml version="1.0" encoding="UTF-8"?>
<d:propfind xmlns:d="DAV:"><d:prop><d:getetag/></d:prop></d:propfind>`

func (p *Provider) listObjectPaths(ctx context.Context, calendarPath string) ([]string, error) {
	target := p.endpoint.ResolveReference(&url.URL{Path: calendarPath})
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", target.String(), strings.NewReader(listObjectsBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("propfind %q: %s", calendarPath, resp.Status)
	}

	var ms struct {
		Responses []struct {
			Href string `xml:"DAV: href"`
		} `xml:"DAV: response"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("decode propfind response: %w", err)
	}

	normalizedCal := strings.TrimSuffix(calendarPath, "/")
	paths := make([]string, 0, len(ms.Responses))
	for _, r := range ms.Responses {
		href := strings.TrimSpace(r.Href)
		if href == "" || strings.HasSuffix(href, "/") {
			continue
		}
		if u, err := url.Parse(href); err == nil {
			href = u.Path
		}
		if unescaped, err := url.PathUnescape(href); err == nil {
			href = unescaped
		}
		if strings.TrimSuffix(href, "/") == normalizedCal {
			continue
		}
		paths = append(paths, href)
	}
	return paths, nil
}

func objectPath(calendarPath, uid string) string {
	if !strings.HasSuffix(calendarPath, "/") {
		calendarPath += "/"
	}
	return calendarPath + uid + ".ics"
}

func (p *Provider) Close() error { return nil }
