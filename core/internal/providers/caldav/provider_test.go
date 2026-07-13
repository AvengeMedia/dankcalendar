package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
)

const principalXML = `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/dav/</d:href>
    <d:propstat>
      <d:prop>
        <d:current-user-principal><d:href>/dav/principals/user/</d:href></d:current-user-principal>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`

const homeSetXML = `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>/dav/principals/user/</d:href>
    <d:propstat>
      <d:prop>
        <c:calendar-home-set><d:href>/dav/calendars/user/</d:href></c:calendar-home-set>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`

func newSecrets(t *testing.T) cal.SecretStore {
	secrets := mocks.NewMockSecretStore(t)
	secrets.EXPECT().Get(context.Background(), "acc", SecretKeyPassword).Return([]byte("pw"), nil)
	return secrets
}

func writeMultiStatus(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	w.Write([]byte(body))
}

// OpenCloud-style server (#38): PROPFIND at the bare host is 405, but the RFC
// 6764 well-known URI redirects to the CalDAV root.
func TestNewFallsBackToWellKnown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		switch {
		case path == "/.well-known/caldav":
			http.Redirect(w, r, "/dav/", http.StatusMovedPermanently)
		case r.Method == "PROPFIND" && path == "/dav":
			writeMultiStatus(w, principalXML)
		case r.Method == "PROPFIND" && path == "/dav/principals/user":
			writeMultiStatus(w, homeSetXML)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	p, err := New(context.Background(), cal.Account{ID: "acc"}, newSecrets(t), server.URL, "user")
	require.NoError(t, err)
	assert.Equal(t, "/dav/calendars/user/", p.homeSet)
}

// A self-signed TLS server stands in for a Nextcloud/self-hosted instance (#50):
// verification fails by default, and SettingInsecureSkipVerify opts out of it.
func TestNewSkipsCertificateVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		switch {
		case r.Method == "PROPFIND" && path == "":
			writeMultiStatus(w, principalXML)
		case r.Method == "PROPFIND" && path == "/dav/principals/user":
			writeMultiStatus(w, homeSetXML)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	_, err := New(context.Background(), cal.Account{ID: "acc"}, newSecrets(t), server.URL, "user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")

	insecure := cal.Account{ID: "acc", Settings: map[string]any{SettingInsecureSkipVerify: true}}
	p, err := New(context.Background(), insecure, newSecrets(t), server.URL, "user")
	require.NoError(t, err)
	assert.Equal(t, "/dav/calendars/user/", p.homeSet)
}

func TestNewReportsPrincipalErrorWithoutWellKnown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/caldav":
			http.NotFound(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	_, err := New(context.Background(), cal.Account{ID: "acc"}, newSecrets(t), server.URL, "user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find caldav principal")
	assert.Contains(t, err.Error(), "405")
}
