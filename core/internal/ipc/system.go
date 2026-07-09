package ipc

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/internal/autostart"
)

func HandleSystem(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "system.autostart.get":
		Respond(w, req.ID, map[string]any{"enabled": autostart.Enabled()})
	case "system.autostart.set":
		if err := setAutostart(ParamBool(req.Params, "enabled")); err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"enabled": autostart.Enabled()})
	case "system.colorScheme.get":
		if deps.ColorScheme == nil {
			Respond(w, req.ID, map[string]any{"available": false, "colorScheme": uint32(0)})
			return
		}
		Respond(w, req.ID, deps.ColorScheme.State())
	case "system.openUri":
		if deps.Opener == nil {
			RespondError(w, req.ID, "uri opener unavailable")
			return
		}
		uri := ParamString(req.Params, "uri")
		if uri == "" {
			RespondError(w, req.ID, "uri is required")
			return
		}
		if err := deps.Opener.OpenURI(uri); err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"opened": true})
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
