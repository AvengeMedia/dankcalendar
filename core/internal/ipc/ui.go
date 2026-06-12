package ipc

import (
	"context"
	"os"
	"strings"
	"syscall"
	"time"
)

func HandleUI(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "ui.show", "ui.hide", "ui.toggle":
		action := strings.TrimPrefix(req.Method, "ui.")
		deps.Bus.Publish("ui", map[string]any{"action": action})
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
