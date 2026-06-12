package ipc

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/internal/autostart"
)

func HandleSystem(_ context.Context, w *ConnWriter, req Request, _ Deps) {
	switch req.Method {
	case "system.autostart.get":
		Respond(w, req.ID, map[string]any{"enabled": autostart.Enabled()})
	case "system.autostart.set":
		if err := setAutostart(ParamBool(req.Params, "enabled")); err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"enabled": autostart.Enabled()})
	default:
		RespondError(w, req.ID, "unknown system method: "+req.Method)
	}
}

func setAutostart(enabled bool) error {
	if enabled {
		return autostart.Enable()
	}
	return autostart.Disable()
}
