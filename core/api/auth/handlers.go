package auth_handler

import (
	"net/http"

	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
)

type CallbackBroker = oauth.CallbackBroker
type CallbackPayload = oauth.CallbackPayload

func NewCallbackBroker() *CallbackBroker {
	return oauth.NewCallbackBroker()
}

func NewCallbackHandler(b *CallbackBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		state := q.Get("state")
		payload := CallbackPayload{
			Code:             q.Get("code"),
			State:            state,
			Error:            q.Get("error"),
			ErrorDescription: q.Get("error_description"),
		}

		switch {
		case state == "":
			http.Error(w, "missing state", http.StatusBadRequest)
			return
		case !b.Deliver(state, payload):
			http.Error(w, "no pending oauth flow for this state", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body style="font-family:sans-serif;text-align:center;padding-top:80px"><h1>Authorization received</h1><p>You can close this window.</p></body></html>`))
	}
}
