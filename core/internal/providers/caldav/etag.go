package caldav

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Some servers (mailbox.org, #81) return ETags without the surrounding
// double-quotes RFC 4918 §15.6 requires, and go-webdav rejects those hard
// (emersion/go-webdav#165). etagNormalizingTransport quotes non-compliant
// ETags in response headers and multistatus bodies before the library sees
// them.
type etagNormalizingTransport struct {
	base http.RoundTripper
}

var getETagRe = regexp.MustCompile(`(<(?:[^:<>/\s]+:)?getetag(?:\s[^>]*)?>)([^<]*)(</(?:[^:<>/\s]+:)?getetag\s*>)`)

func (t etagNormalizingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
