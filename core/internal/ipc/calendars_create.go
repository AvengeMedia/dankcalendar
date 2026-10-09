package ipc

import (
	"context"
	"strings"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/local"
	"github.com/AvengeMedia/dankgo/log"
)

func handleCalendarCreate(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	accountID := ParamString(req.Params, "accountId")
	name := strings.TrimSpace(ParamString(req.Params, "name"))
	switch {
	case accountID == "":
		w.RespondError(req.ID, "accountId is required")
		return
	case name == "":
		w.RespondError(req.ID, "name is required")
		return
	}

	acc, err := deps.Repo.GetAccount(ctx, accountID)
	if err != nil {
		w.RespondError(req.ID, err.Error())
		return
	}
	if string(acc.Kind) != string(calendar.AccountLocal) {
		w.RespondError(req.ID, "only local accounts support creating calendars")
		return
	}
	root, _ := acc.Settings["root"].(string)
	if root == "" {
		w.RespondError(req.ID, "local account is missing its directory")
		return
	}

	cal, err := local.CreateCalendar(root, name)
	if err != nil {
		w.RespondError(req.ID, err.Error())
		return
	}

	// Sync so the new file is discovered and persisted with an id before the
	// UI refreshes; the file already exists, so a failure here is non-fatal.
	if deps.Sync != nil {
		if err := deps.Sync.SyncAccount(ctx, acc); err != nil {
			log.Warnf("sync after calendar create: %v", err)
		}
	}
	if deps.Bus != nil {
		deps.Bus.Publish("calendars", map[string]any{"type": "created", "accountId": accountID})
	}
	w.Respond(req.ID, map[string]any{"accountId": accountID, "name": cal.Name, "remoteId": cal.RemoteID})
}
