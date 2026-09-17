package icsimport

import (
	"fmt"
	"net/url"
	"strings"

	ical "github.com/emersion/go-ical"
)

// SubscriptionURL accepts only the transports supported by the feed provider.
func SubscriptionURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil {
		return "", fmt.Errorf("invalid calendar subscription URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "webcal", "webcals":
		u.Scheme = strings.ToLower(u.Scheme)
		return u.String(), nil
	default:
		return "", fmt.Errorf("calendar subscription requires an HTTP(S) or webcal URL")
	}
}

func sourceOf(cal *ical.Calendar) (string, error) {
	// Scheduling messages remain invitations even when a producer adds SOURCE.
	switch methodOf(cal) {
	case "", "PUBLISH":
	default:
		return "", nil
	}
	// RFC 7986 SOURCE identifies refreshable data; an event URL does not.
	for _, name := range []string{"SOURCE", "X-WR-CALURL"} {
		if p := cal.Props.Get(name); p != nil && strings.TrimSpace(p.Value) != "" {
			return SubscriptionURL(p.Value)
		}
	}
	return "", nil
}
