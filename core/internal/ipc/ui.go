package ipc

import (
	"context"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/duxlabs-dev/dux-calendar/core/internal/icsimport"
	"github.com/duxlabs-dev/dux-calendar/core/internal/paths"
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
				w.RespondError(req.ID, "unknown ui view: "+view)
				return
			}
			payload["view"] = view
		}
		deps.Bus.Publish("ui", payload)
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.open":
		url, err := icsimport.SubscriptionURL(ParamString(req.Params, "url"))
		if err != nil {
			w.RespondError(req.ID, err.Error())
			return
		}
		publishUI(deps, map[string]any{"action": "subscribe", "url": url})
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.openIcs":
		ics := ParamString(req.Params, "ics")
		if strings.TrimSpace(ics) == "" {
			w.RespondError(req.ID, "ui.openIcs requires ics")
			return
		}
		doc, err := icsimport.Parse([]byte(ics))
		if err != nil {
			w.RespondError(req.ID, err.Error())
			return
		}
		if doc.Source != "" {
			publishUI(deps, map[string]any{"action": "subscribe", "url": doc.Source})
			w.Respond(req.ID, map[string]any{"ok": true})
			return
		}
		payload := map[string]any{"action": "importIcs", "ics": ics}
		if name := strings.TrimSpace(ParamString(req.Params, "name")); name != "" {
			payload["name"] = name
		}
		publishUI(deps, payload)
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.openEvent":
		uid := strings.TrimSpace(ParamString(req.Params, "uid"))
		if uid == "" {
			w.RespondError(req.ID, "ui.openEvent requires a uid")
			return
		}
		payload := map[string]any{"action": "openEvent", "uid": uid}
		if start := strings.TrimSpace(ParamString(req.Params, "start")); start != "" {
			payload["start"] = start
		}
		publishUI(deps, payload)
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.newEvent":
		payload := map[string]any{"action": "newEvent"}
		if start := strings.TrimSpace(ParamString(req.Params, "start")); start != "" {
			payload["start"] = start
		}
		publishUI(deps, payload)
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.quit":
		w.Respond(req.ID, map[string]any{"ok": true})
		go func() {
			// Give the response a moment to flush before tearing down.
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		}()
	case "ui.restart":
		path, err := paths.ShowAfterRestartPath()
		if err != nil {
			w.RespondError(req.ID, "ui.restart: "+err.Error())
			return
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			w.RespondError(req.ID, "ui.restart: "+err.Error())
			return
		}
		w.Respond(req.ID, map[string]any{"ok": true})
		go func() {
			time.Sleep(100 * time.Millisecond)
			// The UI supervisor handles SIGUSR1 as a restart.
			_ = syscall.Kill(os.Getpid(), syscall.SIGUSR1)
		}()
	default:
		w.RespondError(req.ID, "unknown ui method: "+req.Method)
	}
}
