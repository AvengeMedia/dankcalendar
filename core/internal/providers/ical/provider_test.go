package ical

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const sampleFeed = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//test//test//EN
X-WR-CALNAME:Test Feed
X-APPLE-CALENDAR-COLOR:#FF2968FF
X-PUBLISHED-TTL:PT2H
BEGIN:VEVENT
UID:evt-1@test
DTSTART:20260507T140000Z
DTEND:20260507T150000Z
SUMMARY:Hello
END:VEVENT
END:VCALENDAR
`

func testProvider(url string) *Provider {
	return New(calendar.Account{ID: "acc"}, url, "", nil, "")
}

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"webcal://example.com/a.ics", "https://example.com/a.ics", true},
		{"https://example.com/a.ics", "https://example.com/a.ics", true},
		{"http://example.com/a.ics", "http://example.com/a.ics", true},
		{"ftp://example.com/a.ics", "", false},
		{"", "", false},
		{"https://", "", false},
	}
	for _, c := range cases {
		got, err := normalizeURL(c.in)
		switch {
		case c.ok && err != nil:
			t.Errorf("normalizeURL(%q) unexpected error: %v", c.in, err)
		case !c.ok && err == nil:
			t.Errorf("normalizeURL(%q) expected error, got %q", c.in, got)
		case c.ok && got != c.want:
			t.Errorf("normalizeURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseISO8601Duration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"PT1H", time.Hour, true},
		{"PT15M", 15 * time.Minute, true},
		{"P1D", 24 * time.Hour, true},
		{"P1W", 7 * 24 * time.Hour, true},
		{"P1DT2H30M", 26*time.Hour + 30*time.Minute, true},
		{"-PT1H", 0, false},
		{"garbage", 0, false},
		{"", 0, false},
		{"P", 0, false},
	}
	for _, c := range cases {
		got, ok := parseISO8601Duration(c.in)
		switch {
		case ok != c.ok:
			t.Errorf("parseISO8601Duration(%q) ok=%v, want %v", c.in, ok, c.ok)
		case ok && got != c.want:
			t.Errorf("parseISO8601Duration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSyncParsesEventsAndTTL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = w.Write([]byte(sampleFeed))
	}))
	defer server.Close()

	p := testProvider(server.URL)
	cal := calendar.Calendar{ID: "cal", RemoteID: remoteID}

	res, err := p.Sync(context.Background(), cal, calendar.SyncCursor{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !res.FullSnapshot {
		t.Error("expected FullSnapshot")
	}
	if len(res.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(res.Changes))
	}
	if got := res.Changes[0].Event.Summary; got != "Hello" {
		t.Errorf("summary = %q, want Hello", got)
	}
	if res.RetryAfter != 2*time.Hour {
		t.Errorf("RetryAfter = %v, want 2h", res.RetryAfter)
	}
	if res.Color != "#FF2968FF" {
		t.Errorf("Color = %q, want #FF2968FF", res.Color)
	}
}

func TestSyncConditionalNotModified(t *testing.T) {
	const etag = `"v1"`
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		_, _ = w.Write([]byte(sampleFeed))
	}))
	defer server.Close()

	p := testProvider(server.URL)
	cal := calendar.Calendar{ID: "cal", RemoteID: remoteID}

	first, err := p.Sync(context.Background(), cal, calendar.SyncCursor{})
	if err != nil {
		t.Fatalf("first Sync: %v", err)
	}
	if len(first.Changes) != 1 {
		t.Fatalf("first sync expected 1 change, got %d", len(first.Changes))
	}

	second, err := p.Sync(context.Background(), cal, first.Cursor)
	if err != nil {
		t.Fatalf("second Sync: %v", err)
	}
	if len(second.Changes) != 0 {
		t.Errorf("second sync expected 0 changes, got %d", len(second.Changes))
	}
	if second.FullSnapshot {
		t.Error("304 result must not be a full snapshot (would prune events)")
	}
	if second.RetryAfter != 2*time.Hour {
		t.Errorf("RetryAfter = %v, want 2h", second.RetryAfter)
	}
	if hits != 2 {
		t.Errorf("server hits = %d, want 2", hits)
	}
}

func TestReadOnly(t *testing.T) {
	p := testProvider("https://example.com/a.ics")
	cal := calendar.Calendar{ID: "cal"}
	ctx := context.Background()

	if _, err := p.CreateEvent(ctx, cal, &calendar.Event{}); err != errReadOnly {
		t.Errorf("CreateEvent err = %v, want %v", err, errReadOnly)
	}
	if _, err := p.UpdateEvent(ctx, cal, &calendar.Event{}); err != errReadOnly {
		t.Errorf("UpdateEvent err = %v, want %v", err, errReadOnly)
	}
	if err := p.DeleteEvent(ctx, cal, calendar.Event{}); err != errReadOnly {
		t.Errorf("DeleteEvent err = %v, want %v", err, errReadOnly)
	}
}
