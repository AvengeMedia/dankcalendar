package ipc

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/internal/autostart"
	"github.com/AvengeMedia/dankcalendar/core/internal/filechooser"
	dankkeyring "github.com/AvengeMedia/dankcalendar/core/internal/keyring"
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
	case "system.pickPath":
		handlePickPath(w, req)
	case "system.isFlatpak":
		Respond(w, req.ID, map[string]any{"flatpak": dankkeyring.InFlatpak()})
	default:
		RespondError(w, req.ID, "unknown system method: "+req.Method)
	}
}

func handlePickPath(w *ConnWriter, req Request) {
	if !filechooser.Available() {
		RespondError(w, req.ID, "file chooser portal only available under Flatpak")
		return
	}
	directory := ParamBool(req.Params, "directory")
	title := ParamString(req.Params, "title")
	paths, err := filechooser.Open(filechooser.Options{
		Title:     title,
		Directory: directory,
		Filters:   ParamStringSlice(req.Params, "filters"),
	})
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	if paths == nil {
		Respond(w, req.ID, map[string]any{"cancelled": true})
		return
	}
	path := ""
	if len(paths) > 0 {
		path = paths[0]
	}
	Respond(w, req.ID, map[string]any{"cancelled": false, "path": path})
}

func setAutostart(enabled bool) error {
	if enabled {
		return autostart.Enable()
	}
	return autostart.Disable()
}
