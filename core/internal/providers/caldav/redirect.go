package caldav

import "net/http"

// net/http rewrites PROPFIND and friends to GET when following 301/302/303
// (golang/go#18570), which strips the body DAV servers need (emersion/go-webdav#123).
// redirectFollowingTransport follows redirects with the original method intact.
type redirectFollowingTransport struct {
	base http.RoundTripper
}

const maxRedirects = 10

func (t redirectFollowingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for hops := 0; ; hops++ {
		resp, err := t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if hops == maxRedirects {
			return resp, nil
		}
		next := redirectedRequest(req, resp)
		if next == nil {
			return resp, nil
		}
		resp.Body.Close()
		req = next
	}
}

func redirectedRequest(req *http.Request, resp *http.Response) *http.Request {
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
	default:
		return nil
	}
	location, err := resp.Location()
	if err != nil || location.Hostname() != req.URL.Hostname() {
		return nil
	}

	next := req.Clone(req.Context())
	next.URL = location
	next.Host = location.Host
	if req.GetBody == nil {
		if req.ContentLength != 0 {
			return nil
		}
		return next
	}
	body, err := req.GetBody()
	if err != nil {
		return nil
	}
	next.Body = body
	return next
}
