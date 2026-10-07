package caldav

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Some servers return properties that go-webdav rejects: mailbox.org (#81)
// sends ETags without the surrounding double-quotes RFC 4918 §15.6 requires
// (emersion/go-webdav#165), and thundermail.com emits getlastmodified dates
// with a single-digit day that http.ParseTime cannot parse (RFC 7231 §3.3.1
// mandates zero-padding). normalizingTransport repairs non-compliant
// responses before the library sees them.
type normalizingTransport struct {
	base http.RoundTripper
}

var getETagRe = regexp.MustCompile(`(<(?:[^:<>/\s]+:)?getetag(?:\s[^>]*)?>)([^<]*)(</(?:[^:<>/\s]+:)?getetag\s*>)`)

var getLastModifiedRe = regexp.MustCompile(`(<(?:[^:<>/\s]+:)?getlastmodified(?:\s[^>]*)?>)([^<]*)(</(?:[^:<>/\s]+:)?getlastmodified\s*>)`)

// nonPaddedDayRe matches a single-digit day-of-month in an HTTP date such as
// "Mon, 7 Sep 2026 21:52:55 GMT".
var nonPaddedDayRe = regexp.MustCompile(`(\b(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun),\s)(\d)\s`)

func (t normalizingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if etag := resp.Header.Get("Etag"); etag != "" {
		resp.Header.Set("Etag", quoteETag(etag))
	}
	if resp.StatusCode != http.StatusMultiStatus {
		return resp, nil
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	body = normalizeETagXML(body)
	body = normalizeLastModifiedXML(body)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	if resp.Header.Get("Content-Length") != "" {
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}
	return resp, nil
}

func normalizeETagXML(body []byte) []byte {
	return getETagRe.ReplaceAllFunc(body, func(m []byte) []byte {
		sub := getETagRe.FindSubmatch(m)

		etag := string(sub[2])
		trimmed := strings.TrimSpace(etag)
		// The ETAG may be quoted with entity encoded quotes which are valid according
		// to the XML standard and go-webdav properly supports them. As such they must
		// also be considered as quoted and skipped, otherwise they will end up double
		// quoted and error during processing.
		quoted := ""
		if strings.HasPrefix(trimmed, "&quot;") && strings.HasSuffix(trimmed, "&quot;") {
			quoted = etag
		} else {
			quoted = quoteETag(etag)
		}

		return append(append(sub[1], quoted...), sub[3]...)
	})
}

func normalizeLastModifiedXML(body []byte) []byte {
	return getLastModifiedRe.ReplaceAllFunc(body, func(m []byte) []byte {
		sub := getLastModifiedRe.FindSubmatch(m)
		padded := nonPaddedDayRe.ReplaceAll(sub[2], []byte(`${1}0${2} `))
		return append(append(sub[1], padded...), sub[3]...)
	})
}

func quoteETag(etag string) string {
	trimmed := strings.TrimSpace(etag)
	if trimmed == "" {
		return etag
	}
	if _, err := strconv.Unquote(trimmed); err == nil {
		return etag
	}
	return strconv.Quote(trimmed)
}
