package ipc

import "context"

func HandleReminders(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	if deps.Reminders == nil {
		RespondError(w, req.ID, "reminders engine unavailable")
		return
	}

	switch req.Method {
	case "reminders.upcoming":
		limit := ParamInt(req.Params, "limit")
		if limit <= 0 {
			limit = 20
		}
		items, err := deps.Reminders.Upcoming(ctx, limit)
		if err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"reminders": items})
	case "reminders.test":
		if err := deps.Reminders.SendTest(); err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"sent": true})
	default:
		RespondError(w, req.ID, "unknown reminders method: "+req.Method)
	}
}
