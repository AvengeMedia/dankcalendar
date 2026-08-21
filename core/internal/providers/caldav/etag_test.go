package caldav

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteETag(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"quoted", `"abc123"`, `"abc123"`},
		{"unquoted", `abc123`, `"abc123"`},
		{"weak", `W/"abc123"`, `"W/\"abc123\""`},
		{"empty", ``, ``},
		{"whitespace only", `  `, `  `},
		{"padded unquoted", ` 1234-5678 `, `"1234-5678"`},
		{"entity encoded (does not apply)", ` &quot;1234-5678&quot; `, `"&quot;1234-5678&quot;"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, quoteETag(tc.in))
		})
	}
}

func TestNormalizeETagXML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"unquoted namespaced",
			`<D:getetag>1234-5678</D:getetag>`,
			`<D:getetag>"1234-5678"</D:getetag>`,
		},
		{
			"unquoted unprefixed",
			`<getetag>1234</getetag>`,
			`<getetag>"1234"</getetag>`,
		},
		{
			"quoted left alone",
			`<d:getetag>"abc"</d:getetag>`,
			`<d:getetag>"abc"</d:getetag>`,
		},
		{
			"entity encoded left alone",
			`<d:getetag>&quot;abc&quot;</d:getetag>`,
			`<d:getetag>&quot;abc&quot;</d:getetag>`,
		},
		{
			"empty left alone",
			`<d:getetag></d:getetag>`,
			`<d:getetag></d:getetag>`,
		},
		{
			"self-closing left alone",
			`<d:getetag/>`,
			`<d:getetag/>`,
		},
		{
			"other elements untouched",
			`<d:displayname>no "quotes" added</d:displayname>`,
			`<d:displayname>no "quotes" added</d:displayname>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, string(normalizeETagXML([]byte(tc.in))))
		})
	}
}

// mailbox.org returns getetag values without the quotes RFC 4918 requires
// (#81); the transport must repair both multistatus bodies and ETag headers.
func TestETagNormalizingTransport(t *testing.T) {
	const multistatus = `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/caldav/cal/event.ics</d:href>
    <d:propstat>
      <d:prop><d:getetag>1755-372673</d:getetag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "REPORT":
			writeMultiStatus(w, multistatus)
		default:
			w.Header().Set("Etag", "1755-372673")
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer server.Close()

	client := &http.Client{Transport: etagNormalizingTransport{base: http.DefaultTransport}}

	req, err := http.NewRequest("REPORT", server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Contains(t, string(body), `<d:getetag>"1755-372673"</d:getetag>`)

	resp, err = client.Do(&http.Request{Method: "PUT", URL: resp.Request.URL})
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, `"1755-372673"`, resp.Header.Get("Etag"))
}

// Nextcloud returns getetag values with quotes entity encoded; the transport
// must not modify multistatus bodies if this is the case.
func TestETagNormalizingTransportEntityEncoded(t *testing.T) {
	const multistatus = `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/caldav/cal/event.ics</d:href>
    <d:propstat>
      <d:prop><d:getetag>&quot;1755-372673&quot;</d:getetag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeMultiStatus(w, multistatus)
	}))
	defer server.Close()

	client := &http.Client{Transport: etagNormalizingTransport{base: http.DefaultTransport}}

	req, err := http.NewRequest("REPORT", server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Contains(t, string(body), `<d:getetag>&quot;1755-372673&quot;</d:getetag>`)
}
