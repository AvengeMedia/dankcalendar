package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

const (
	graphBase      = "https://graph.microsoft.com/v1.0"
	snapshotPrefix = "snapshot "
)

type Provider struct {
	account    cal.Account
	secrets    cal.SecretStore
	httpClient *http.Client
	cfg        *oauth2.Config
	masters    map[string]graphEvent
}

func New(ctx context.Context, account cal.Account, secrets cal.SecretStore) (*Provider, error) {
	appBytes, err := secrets.Get(ctx, account.ID, SecretKeyApp)
	if err != nil {
		return nil, fmt.Errorf("missing microsoft app credentials for account %q: %w", account.ID, err)
	}

	var creds oauth.MicrosoftAppCredentials
	if err := json.Unmarshal(appBytes, &creds); err != nil {
		return nil, fmt.Errorf("decode microsoft app credentials: %w", err)
	}

	tokenBytes, err := secrets.Get(ctx, account.ID, SecretKeyToken)
	if err != nil {
		return nil, fmt.Errorf("account %q is not authorized; run `dcal account microsoft login %s`", account.ID, account.ID)
	}

	tok, err := oauth.UnmarshalToken(tokenBytes)
	if err != nil {
		return nil, err
	}

	cfg := oauth.MicrosoftConfig(creds, "")
	tokenSource := persistingTokenSource{
		base:      cfg.TokenSource(ctx, tok),
		secrets:   secrets,
		accountID: account.ID,
	}

	return &Provider{
		account:    account,
		secrets:    secrets,
		httpClient: oauth2.NewClient(ctx, &tokenSource),
		cfg:        cfg,
		masters:    make(map[string]graphEvent),
	}, nil
}

func (p *Provider) Kind() cal.AccountKind { return cal.AccountMicrosoft }
func (p *Provider) Account() cal.Account  { return p.account }

type graphError struct {
	Status  int
	Code    string
	Message string
}

func (e *graphError) Error() string {
	return fmt.Sprintf("graph api error %d (%s): %s", e.Status, e.Code, e.Message)
}

func (p *Provider) doJSON(ctx context.Context, method, reqURL string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Prefer", `outlook.timezone="UTC"`)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusBadRequest {
		var ge struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		_ = json.Unmarshal(data, &ge)
		return &graphError{Status: res.StatusCode, Code: ge.Error.Code, Message: ge.Error.Message}
	}

	if res.StatusCode == http.StatusNoContent || out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

type graphCalendar struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HexColor string `json:"hexColor"`
	CanEdit  bool   `json:"canEdit"`
}

func (p *Provider) ListCalendars(ctx context.Context) ([]cal.Calendar, error) {
	var out []cal.Calendar
	next := graphBase + "/me/calendars?$top=100"
	for next != "" {
		var page struct {
			Value    []graphCalendar `json:"value"`
			NextLink string          `json:"@odata.nextLink"`
		}
		if err := p.doJSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, fmt.Errorf("list microsoft calendars: %w", err)
		}

		for _, item := range page.Value {
			c := cal.Calendar{
				AccountID: p.account.ID,
				RemoteID:  item.ID,
				Name:      item.Name,
				ReadOnly:  !item.CanEdit,
			}
			if item.HexColor != "" && item.HexColor != "auto" {
				c.Color = item.HexColor
			}
			out = append(out, c)
		}
		next = page.NextLink
	}
	return out, nil
}

type graphEventPage struct {
	Value     []graphEvent `json:"value"`
	NextLink  string       `json:"@odata.nextLink"`
	DeltaLink string       `json:"@odata.deltaLink"`
}

func (p *Provider) Sync(ctx context.Context, c cal.Calendar, cursor cal.SyncCursor) (*cal.SyncResult, error) {
	reqURL, snapshot := deltaURL(c, cursor)

	var page graphEventPage
	if err := p.doJSON(ctx, http.MethodGet, reqURL, nil, &page); err != nil {
		var ge *graphError
		if errors.As(err, &ge) && ge.Status == http.StatusGone && cursor.Token != "" {
			return p.Sync(ctx, c, cal.SyncCursor{CalendarID: cursor.CalendarID})
		}
		return nil, fmt.Errorf("microsoft delta sync: %w", err)
	}

	result := &cal.SyncResult{
		Cursor:       cal.SyncCursor{CalendarID: cursor.CalendarID},
		FullSnapshot: snapshot,
	}
	for _, item := range page.Value {
		switch {
		case item.Removed != nil:
			result.Changes = append(result.Changes, cal.EventChange{Type: cal.ChangeDelete, RemoteID: item.ID})
		case item.Type == "seriesMaster":
			// Graph already expands the series into occurrences; the master
			// is only kept around to fill in their trimmed-down properties.
			p.masters[item.ID] = item
		default:
			ev := graphToEvent(p.hydrateOccurrence(ctx, item))
			result.Changes = append(result.Changes, cal.EventChange{Type: cal.ChangeUpsert, Event: &ev})
		}
	}

	switch {
	case page.NextLink != "":
		result.More = true
		result.Cursor.Token = page.NextLink
		if snapshot {
			result.Cursor.Token = snapshotPrefix + page.NextLink
		}
	case page.DeltaLink != "":
		result.Cursor.Token = page.DeltaLink
	}
	return result, nil
}

// hydrateOccurrence fills occurrence/exception items from their series
// master: delta responses trim occurrences down to little more than id,
// times, and seriesMasterId.
func (p *Provider) hydrateOccurrence(ctx context.Context, item graphEvent) graphEvent {
	if item.SeriesMasterID == "" || item.Subject != "" {
		return item
	}

	master, ok := p.masters[item.SeriesMasterID]
	if !ok {
		fetched, err := p.fetchEvent(ctx, item.SeriesMasterID)
		if err != nil {
			return item
		}
		p.masters[item.SeriesMasterID] = fetched
		master = fetched
	}

	item.Subject = master.Subject
	if item.BodyPreview == "" {
		item.BodyPreview = master.BodyPreview
	}
	if item.Location == nil {
		item.Location = master.Location
	}
	if item.WebLink == "" {
		item.WebLink = master.WebLink
	}
	if item.Organizer == nil {
		item.Organizer = master.Organizer
	}
	if len(item.Attendees) == 0 {
		item.Attendees = master.Attendees
	}
	if item.ShowAs == "" {
		item.ShowAs = master.ShowAs
	}
	if item.Sensitivity == "" {
		item.Sensitivity = master.Sensitivity
	}
	return item
}

func (p *Provider) fetchEvent(ctx context.Context, id string) (graphEvent, error) {
	var out graphEvent
	err := p.doJSON(ctx, http.MethodGet, graphBase+"/me/events/"+url.PathEscape(id), nil, &out)
	return out, err
}

func deltaURL(c cal.Calendar, cursor cal.SyncCursor) (reqURL string, snapshot bool) {
	switch {
	case cursor.Token == "":
		now := time.Now().UTC()
		q := url.Values{
			"startDateTime": {now.AddDate(-1, 0, 0).Format(time.RFC3339)},
			"endDateTime":   {now.AddDate(2, 0, 0).Format(time.RFC3339)},
		}
		return graphBase + "/me/calendars/" + url.PathEscape(c.RemoteID) + "/calendarView/delta?" + q.Encode(), true
	case strings.HasPrefix(cursor.Token, snapshotPrefix):
		return strings.TrimPrefix(cursor.Token, snapshotPrefix), true
	default:
		return cursor.Token, false
	}
}

func (p *Provider) ListEvents(ctx context.Context, c cal.Calendar, opts cal.ListEventsOptions) ([]cal.Event, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 250
	}

	now := time.Now().UTC()
	start, end := opts.Start, opts.End
	if start.IsZero() {
		start = now.AddDate(-1, 0, 0)
	}
	if end.IsZero() {
		end = now.AddDate(2, 0, 0)
	}

	q := url.Values{
		"startDateTime": {start.UTC().Format(time.RFC3339)},
		"endDateTime":   {end.UTC().Format(time.RFC3339)},
		"$top":          {strconv.Itoa(limit)},
	}
	next := graphBase + "/me/calendars/" + url.PathEscape(c.RemoteID) + "/calendarView?" + q.Encode()

	var out []cal.Event
	for next != "" && len(out) < limit {
		var page graphEventPage
		if err := p.doJSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, fmt.Errorf("list microsoft events: %w", err)
		}
		for _, item := range page.Value {
			if len(out) >= limit {
				break
			}
			out = append(out, graphToEvent(item))
		}
		next = page.NextLink
	}
	return out, nil
}

func (p *Provider) CreateEvent(ctx context.Context, c cal.Calendar, ev *cal.Event) (*cal.Event, error) {
	var created graphEvent
	reqURL := graphBase + "/me/calendars/" + url.PathEscape(c.RemoteID) + "/events"
	if err := p.doJSON(ctx, http.MethodPost, reqURL, eventToGraph(ev), &created); err != nil {
		return nil, fmt.Errorf("microsoft create event: %w", err)
	}

	out := graphToEvent(created)
	out.ID = ev.ID
	out.CalendarID = ev.CalendarID
	return &out, nil
}

func (p *Provider) UpdateEvent(ctx context.Context, c cal.Calendar, ev *cal.Event) (*cal.Event, error) {
	if ev.RemoteID == "" {
		return nil, errors.New("microsoft update event: missing remote id")
	}

	var updated graphEvent
	reqURL := graphBase + "/me/events/" + url.PathEscape(ev.RemoteID)
	if err := p.doJSON(ctx, http.MethodPatch, reqURL, eventToGraph(ev), &updated); err != nil {
		return nil, fmt.Errorf("microsoft update event: %w", err)
	}

	out := graphToEvent(updated)
	out.ID = ev.ID
	out.CalendarID = ev.CalendarID
	return &out, nil
}

func (p *Provider) DeleteEvent(ctx context.Context, c cal.Calendar, ev cal.Event) error {
	if ev.RemoteID == "" {
		return errors.New("microsoft delete event: missing remote id")
	}

	reqURL := graphBase + "/me/events/" + url.PathEscape(ev.RemoteID)
	err := p.doJSON(ctx, http.MethodDelete, reqURL, nil, nil)
	var ge *graphError
	if errors.As(err, &ge) && ge.Status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("microsoft delete event: %w", err)
	}
	return nil
}

func (p *Provider) Close() error { return nil }

type persistingTokenSource struct {
	base      oauth2.TokenSource
	secrets   cal.SecretStore
	accountID string
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	data, err := oauth.MarshalToken(tok)
	if err != nil {
		return tok, nil
	}
	if err := s.secrets.Set(context.Background(), s.accountID, SecretKeyToken, data); err != nil {
		return tok, nil
	}
	return tok, nil
}
