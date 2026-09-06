package httpauth

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var digestFieldRe = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|([^\s,]+))`)

// digestServer validates like sabre/http: MD5, qop=auth, uri taken from the
// Authorization header, no nonce bookkeeping.
type digestServer struct {
	password   string
	nonce      atomic.Value
	requests   atomic.Int32
	challenges atomic.Int32
	bodies     chan string
}

func newDigestServer(password string) *digestServer {
	s := &digestServer{password: password, bodies: make(chan string, 16)}
	s.nonce.Store("nonce-1")
	return s
}

func (s *digestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)
	body, _ := io.ReadAll(r.Body)
	s.bodies <- string(body)
	if !s.valid(r) {
		s.challenges.Add(1)
		w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Digest realm="SabreDAV", qop="auth", nonce="%s", opaque="op"`, s.nonce.Load()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *digestServer) valid(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Digest ") {
		return false
	}
	parts := map[string]string{}
	for _, m := range digestFieldRe.FindAllStringSubmatch(auth, -1) {
		parts[m[1]] = m[2] + m[3]
	}
	if parts["nonce"] != s.nonce.Load() {
		return false
	}
	a1 := md5hex(parts["username"] + ":SabreDAV:" + s.password)
	a2 := md5hex(r.Method + ":" + parts["uri"])
	expected := md5hex(strings.Join([]string{a1, parts["nonce"], parts["nc"], parts["cnonce"], parts["qop"], a2}, ":"))
	return parts["response"] == expected && parts["opaque"] == "op"
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func propfind(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("PROPFIND", url, strings.NewReader("<propfind/>"))
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	return resp
}

func TestTransportAnswersDigestChallengeThenSignsPreemptively(t *testing.T) {
	backend := newDigestServer("pw")
	server := httptest.NewServer(backend)
	defer server.Close()
	client := &http.Client{Transport: &Transport{Username: "user", Password: "pw"}}

	resp := propfind(t, client, server.URL+"/dav.php/?x=1")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), backend.requests.Load())
	assert.Equal(t, "<propfind/>", <-backend.bodies)
	assert.Equal(t, "<propfind/>", <-backend.bodies)

	resp = propfind(t, client, server.URL+"/dav.php/calendars/")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), backend.requests.Load())
	assert.Equal(t, int32(1), backend.challenges.Load())
}

func TestTransportRetriesOnceWithNewNonce(t *testing.T) {
	backend := newDigestServer("pw")
	server := httptest.NewServer(backend)
	defer server.Close()
	client := &http.Client{Transport: &Transport{Username: "user", Password: "pw"}}

	propfind(t, client, server.URL+"/")
	backend.nonce.Store("nonce-2")
	resp := propfind(t, client, server.URL+"/")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(4), backend.requests.Load())
}

func TestTransportGivesUpOnRejectedCredentials(t *testing.T) {
	backend := newDigestServer("other")
	server := httptest.NewServer(backend)
	defer server.Close()
	client := &http.Client{Transport: &Transport{Username: "user", Password: "pw"}}

	resp := propfind(t, client, server.URL+"/")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, int32(2), backend.requests.Load())

	resp = propfind(t, client, server.URL+"/")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, int32(3), backend.requests.Load())
}

func TestTransportKeepsBasicForBasicServers(t *testing.T) {
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		if _, _, ok := r.BasicAuth(); !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="dav"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := &http.Client{Transport: &Transport{Username: "user", Password: "pw"}}

	resp := propfind(t, client, server.URL+"/")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, authorizations, 1)
	assert.True(t, strings.HasPrefix(authorizations[0], "Basic "))
}
