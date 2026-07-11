package ipc

import (
	"context"
	"os"
	"strings"
	"syscall"
	"time"
)

var uiViews = []string{"month", "week", "day", "agenda"}

func validUIView(view string) bool {
	for _, v := range uiViews {
		if v == view {
			return true
		}
	}
	return false
}

// publishUI delivers a "ui" payload live when the GUI is listening, else stashes
// it for the next "ui" subscriber so a cold-started window still honors it.
func publishUI(deps Deps, payload map[string]any) {
	if deps.Bus.HasSubscriber("ui") {
		deps.Bus.Publish("ui", payload)
		return
	}
	if deps.Pending != nil {
		deps.Pending.Set(payload)
	}
}

func HandleUI(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "ui.show", "ui.hide", "ui.toggle":
		action := strings.TrimPrefix(req.Method, "ui.")
		payload := map[string]any{"action": action}
		view := strings.TrimSpace(ParamString(req.Params, "view"))
		if view != "" {
			if !validUIView(view) {
				RespondError(w, req.ID, "unknown ui view: "+view)
				return
			}
			payload["view"] = view
		}
		deps.Bus.Publish("ui", payload)
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.open":
		url := strings.TrimSpace(ParamString(req.Params, "url"))
		if url == "" {
			RespondError(w, req.ID, "ui.open requires a url")
			return
		}
		publishUI(deps, map[string]any{"action": "subscribe", "url": url})
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.openEvent":
		uid := strings.TrimSpace(ParamString(req.Params, "uid"))
		if uid == "" {
			RespondError(w, req.ID, "ui.openEvent requires a uid")
			return
		}
		payload := map[string]any{"action": "openEvent", "uid": uid}
		if start := strings.TrimSpace(ParamString(req.Params, "start")); start != "" {
			payload["start"] = start
		}
		publishUI(deps, payload)
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.newEvent":
		payload := map[string]any{"action": "newEvent"}
		if start := strings.TrimSpace(ParamString(req.Params, "start")); start != "" {
			payload["start"] = start
		}
		publishUI(deps, payload)
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.quit":
		Respond(w, req.ID, map[string]any{"ok": true})
		go func() {
			// Give the response a moment to flush before tearing down.
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		}()
	default:
		RespondError(w, req.ID, "unknown ui method: "+req.Method)
	}
}
