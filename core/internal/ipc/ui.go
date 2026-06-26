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
		// Deliver live when the GUI is listening, else stash it for the next
		// "ui" subscriber so a cold-started window still opens the link.
		if deps.Bus.HasSubscriber("ui") {
			deps.Bus.Publish("ui", map[string]any{"action": "subscribe", "url": url})
		} else if deps.Pending != nil {
			deps.Pending.Set(url)
		}
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
