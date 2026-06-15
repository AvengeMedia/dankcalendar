package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

type CallbackResult struct {
	Code  string
	State string
}

type LoopbackFlow struct {
	listener     net.Listener
	server       *http.Server
	resultCh     chan CallbackResult
	errCh        chan error
	state        string
	verifier     string
	redirectHost string
	redirectPath string
}

func StartLoopback(ctx context.Context, addr string) (*LoopbackFlow, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("loopback listen: %w", err)
	}

	state, err := randomToken(32)
	if err != nil {
		listener.Close()
		return nil, err
	}

	flow := &LoopbackFlow{
		listener: listener,
		resultCh: make(chan CallbackResult, 1),
		errCh:    make(chan error, 1),
		state:    state,
		verifier: oauth2.GenerateVerifier(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", flow.handleCallback)
	mux.HandleFunc("/", flow.handleCallback)

	flow.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := flow.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			flow.errCh <- err
		}
	}()
	return flow, nil
}

func (f *LoopbackFlow) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if errParam := q.Get("error"); errParam != "" {
		msg := FormatCallbackError(errParam, q.Get("error_description"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>Authorization failed</h1><p>%s</p><p>You can close this window.</p>", html.EscapeString(msg))
		f.errCh <- errors.New(msg)
		return
	}

	code := q.Get("code")
	state := q.Get("state")

	switch {
	case code == "":
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	case state != f.state:
		http.Error(w, "state mismatch", http.StatusBadRequest)
		f.errCh <- errors.New("oauth state mismatch")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html><html><head><title>Dank Calendar</title></head><body style="font-family:sans-serif;text-align:center;padding-top:80px"><h1>Authorization successful</h1><p>You can close this window and return to the terminal.</p></body></html>`)
	f.resultCh <- CallbackResult{Code: code, State: state}
}

func (f *LoopbackFlow) RedirectURL() string {
	addr := f.listener.Addr().(*net.TCPAddr)
	host := f.redirectHost
	if host == "" {
		host = "127.0.0.1"
	}
	path := f.redirectPath
	if path == "" {
		path = "/oauth/callback"
	}
	u := url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", host, addr.Port), Path: path}
	return u.String()
}

func (f *LoopbackFlow) State() string    { return f.state }
func (f *LoopbackFlow) Verifier() string { return f.verifier }

func (f *LoopbackFlow) Wait(ctx context.Context, timeout time.Duration) (CallbackResult, error) {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-f.resultCh:
		return res, nil
	case err := <-f.errCh:
		return CallbackResult{}, err
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	case <-timer.C:
		return CallbackResult{}, errors.New("oauth callback timed out")
	}
}

func (f *LoopbackFlow) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return f.server.Shutdown(shutdownCtx)
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
