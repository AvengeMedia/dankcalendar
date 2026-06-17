package ipc

import (
	"context"
	"strings"
)

func Route(ctx context.Context, w *ConnWriter, req Request, deps Deps, sub *Subscriber) {
	switch req.Method {
	case "ping":
		Respond(w, req.ID, map[string]any{"pong": true})
		return
	case "version":
		Respond(w, req.ID, map[string]any{"version": deps.Version, "apiVersion": APIVersion})
		return
	case "describe":
		Respond(w, req.ID, map[string]any{"apiVersion": APIVersion, "methods": Methods})
		return
	case "subscribe":
		HandleSubscribe(w, req, deps, sub)
		return
	case "unsubscribe":
		HandleUnsubscribe(w, req, sub)
		return
	}

	switch {
	case strings.HasPrefix(req.Method, "accounts."):
		HandleAccounts(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "calendars."):
		HandleCalendars(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "events."):
		HandleEvents(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "reminders."):
		HandleReminders(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "ui."):
		HandleUI(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "system."):
		HandleSystem(ctx, w, req, deps)
	default:
		RespondError(w, req.ID, "unknown method: "+req.Method)
	}
}
