package ical

import (
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

	ical "github.com/emersion/go-ical"
)

const (
	userAgent      = "dankcalendar"
	maxBodyBytes   = 25 << 20
	requestTimeout = 30 * time.Second

	defaultTTL = time.Hour
	minTTL     = 15 * time.Minute
	maxTTL     = 24 * time.Hour
)

var httpClient = &http.Client{Timeout: requestTimeout}

// normalizeURL rewrites webcal:// to https:// and rejects non-http(s) schemes.
func normalizeURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("url is required")
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	switch strings.ToLower(u.Scheme) {
	case "webcal":
		u.Scheme = "https"
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported url scheme %q (expected http, https or webcal)", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("url has no host")
	}
	return u.String(), nil
}

// cursor persists between syncs in the calendar sync_token so the next fetch can
// be conditional and reuse the feed's published TTL.
type cursor struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	TTLSeconds   int64  `json:"ttlSeconds,omitempty"`
}

func decodeCursor(token string) cursor {
	var c cursor
	if token == "" {
		return c
	}
	_ = json.Unmarshal([]byte(token), &c)
	return c
}

func (c cursor) encode() string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(data)
}

func (c cursor) ttl() time.Duration { return clampTTL(time.Duration(c.TTLSeconds) * time.Second) }

func clampTTL(d time.Duration) time.Duration {
	switch {
	case d <= 0:
		return defaultTTL
	case d < minTTL:
		return minTTL
	case d > maxTTL:
		return maxTTL
	default:
		return d
	}
}

type feed struct {
	doc         *ical.Calendar
	name        string
	cursor      cursor
	notModified bool
}

func fetch(ctx context.Context, feedURL, username string, password []byte, prev cursor) (*feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/calendar, text/plain;q=0.9, */*;q=0.5")
	if username != "" {
		req.SetBasicAuth(username, string(password))
	}
	if prev.ETag != "" {
		req.Header.Set("If-None-Match", prev.ETag)
	}
	if prev.LastModified != "" {
		req.Header.Set("If-Modified-Since", prev.LastModified)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return &feed{notModified: true, cursor: prev}, nil
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("fetch feed: %s", resp.Status)
	}

	doc, err := ical.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode()
	if err != nil {
		return nil, fmt.Errorf("decode feed: %w", err)
	}

	return &feed{
		doc:  doc,
		name: calendarName(doc),
		cursor: cursor{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
			TTLSeconds:   int64(feedTTL(doc).Seconds()),
		},
	}, nil
}

func calendarName(doc *ical.Calendar) string {
	if p := doc.Props.Get("X-WR-CALNAME"); p != nil {
		return strings.TrimSpace(p.Value)
	}
	return ""
}

func feedTTL(doc *ical.Calendar) time.Duration {
	for _, name := range []string{"REFRESH-INTERVAL", "X-PUBLISHED-TTL"} {
		p := doc.Props.Get(name)
		if p == nil {
			continue
		}
		if d, ok := parseISO8601Duration(p.Value); ok {
			return d
		}
	}
	return defaultTTL
}

// parseISO8601Duration handles the iCalendar duration subset (PnW / PnDTnHnMnS).
func parseISO8601Duration(value string) (time.Duration, bool) {
	s := strings.TrimSpace(value)
	switch {
	case s == "":
		return 0, false
	case s[0] == '+':
		s = s[1:]
	case s[0] == '-':
		return 0, false
	}
	if s == "" || s[0] != 'P' {
		return 0, false
	}
	s = s[1:]

	var total time.Duration
	var num string
	inTime, matched := false, false
	for _, r := range s {
		switch {
		case r == 'T':
			inTime = true
		case r >= '0' && r <= '9':
			num += string(r)
		default:
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, false
			}
			unit, ok := durationUnit(r, inTime)
			if !ok {
				return 0, false
			}
			total += time.Duration(n) * unit
			num, matched = "", true
		}
	}
	if num != "" || !matched {
		return 0, false
	}
	return total, true
}

func durationUnit(r rune, inTime bool) (time.Duration, bool) {
	switch {
	case r == 'W':
		return 7 * 24 * time.Hour, true
	case r == 'D':
		return 24 * time.Hour, true
	case r == 'H' && inTime:
		return time.Hour, true
	case r == 'M' && inTime:
		return time.Minute, true
	case r == 'S' && inTime:
		return time.Second, true
	default:
		return 0, false
	}
}
