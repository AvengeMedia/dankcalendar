package icsimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionSource(t *testing.T) {
	for _, tc := range []struct{ name, method, property, want string }{
		{"source", "", "SOURCE;VALUE=URI:https://example.com/feed.ics", "https://example.com/feed.ics"},
		{"legacy source", "PUBLISH", "X-WR-CALURL:webcal://example.com/feed", "webcal://example.com/feed"},
		{"invitation", "REQUEST", "SOURCE:https://example.com/feed", ""},
		{"event page", "", "URL:https://example.com/meeting", ""},
		{"recurrence is not subscription", "", "RRULE:FREQ=DAILY", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := strings.Replace(invitation, "METHOD:REQUEST", "METHOD:"+tc.method+"\r\n"+tc.property, 1)
			doc, err := Parse([]byte(input))
			require.NoError(t, err)
			assert.Equal(t, tc.want, doc.Source)
		})
	}
	doc, err := Parse([]byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nSOURCE:https://example.com/feed\r\nEND:VCALENDAR\r\n"))
	require.NoError(t, err)
	assert.Empty(t, doc.Events)
	assert.Equal(t, "https://example.com/feed", doc.Source)
}

func TestSubscriptionURL(t *testing.T) {
	for _, raw := range []string{"file:///tmp/a.ics", "https:///a", "javascript:alert(1)", "https://me:secret@example.com/feed", "%"} {
		_, err := SubscriptionURL(raw)
		assert.Error(t, err, raw)
	}
	got, err := SubscriptionURL(" WEBCALS://example.com/feed ")
	require.NoError(t, err)
	assert.Equal(t, "webcals://example.com/feed", got)
}
