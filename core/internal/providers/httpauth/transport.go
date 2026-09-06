package httpauth

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Transport authenticates outgoing requests. It sends Basic credentials until
// the server answers with a Digest challenge (RFC 7616), then keeps signing
// with Digest for the rest of the session.
type Transport struct {
	Base     http.RoundTripper
	Username string
	Password string

	mu         sync.Mutex
	challenge  *challenge
	nonceCount uint32
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	first := t.authorize(req)
	resp, err := t.base().RoundTrip(first)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	ch := parseChallenge(resp.Header.Values("WWW-Authenticate"))
	if ch == nil || t.rejected(first, ch) {
		return resp, nil
	}
	retry := replayable(req)
	if retry == nil {
		return resp, nil
	}

	drain(resp)
	t.setChallenge(ch)
	return t.base().RoundTrip(t.authorize(retry))
}

func (t *Transport) base() http.RoundTripper {
	if t.Base == nil {
		return http.DefaultTransport
	}
	return t.Base
}

func (t *Transport) authorize(req *http.Request) *http.Request {
	out := req.Clone(req.Context())
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.challenge == nil {
		out.SetBasicAuth(t.Username, t.Password)
		return out
	}
	t.nonceCount++
	out.Header.Set("Authorization", t.challenge.authorization(t.Username, t.Password, req.Method, req.URL.RequestURI(), t.nonceCount, cnonce()))
	return out
}

// rejected reports whether the server refused a Digest response for the same
// nonce without marking it stale, which means the credentials are wrong.
func (t *Transport) rejected(sent *http.Request, ch *challenge) bool {
	if !strings.HasPrefix(sent.Header.Get("Authorization"), "Digest ") {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.challenge != nil && t.challenge.nonce == ch.nonce && !ch.stale
}

func (t *Transport) setChallenge(ch *challenge) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.challenge == nil || t.challenge.nonce != ch.nonce {
		t.nonceCount = 0
	}
	t.challenge = ch
}

func replayable(req *http.Request) *http.Request {
	if req.GetBody == nil {
		if req.Body != nil && req.Body != http.NoBody {
			return nil
		}
		return req
	}
	body, err := req.GetBody()
	if err != nil {
		return nil
	}
	next := req.Clone(req.Context())
	next.Body = body
	return next
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
}

func cnonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
