package ipc

import (
	"context"
	"fmt"
	"slices"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/icsimport"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func parseIcsParam(params map[string]any) (*icsimport.Document, error) {
	ics := ParamString(params, "ics")
	if ics == "" {
		return nil, fmt.Errorf("ics is required")
	}
	return icsimport.Parse([]byte(ics))
}

func handleEventsParseIcs(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	doc, err := parseIcsParam(req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	items := make([]map[string]any, 0, len(doc.Events))
	for i := range doc.Events {
		ev := &doc.Events[i]
		entry := map[string]any{"event": ev}
		existing, err := deps.Repo.GetEventByUID(ctx, ev.UID, "")
		switch {
		case err == nil:
			entry["existing"] = mapEvent(existing)
		case !repo.IsNotFound(err):
			RespondError(w, req.ID, err.Error())
			return
		}
		items = append(items, entry)
	}
	Respond(w, req.ID, map[string]any{"method": doc.Method, "events": items})
}

func handleEventsImportIcs(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	calendarID := ParamString(req.Params, "calendarId")
	if calendarID == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}
	doc, err := parseIcsParam(req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	selected := selectEvents(doc.Events, ParamStringSlice(req.Params, "uids"))
	if len(selected) == 0 {
		RespondError(w, req.ID, "no matching events to import")
		return
	}

	provider, domCal, err := providerForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()
	if !domCal.HoldsEvents() {
		RespondError(w, req.ID, fmt.Sprintf("calendar %q holds tasks, not events", domCal.Name))
		return
	}

	results := make([]map[string]any, 0, len(selected))
	imported := 0
	for i := range selected {
		ev := &selected[i]
		existing, err := deps.Repo.FindEventByUID(ctx, domCal.ID, ev.UID)
		switch {
		case err == nil:
			results = append(results, map[string]any{"event": mapEvent(existing), "existing": true})
			continue
		case !repo.IsNotFound(err):
			RespondError(w, req.ID, err.Error())
			return
		}

		created, err := icsimport.Create(ctx, provider, domCal, ev)
		if err != nil {
			respondImportError(w, req, deps, domCal.ID, imported, fmt.Sprintf("import %q: %v", ev.Summary, err))
			return
		}
		stored, err := persistEvent(ctx, deps, domCal.ID, created)
		if err != nil {
			respondImportError(w, req, deps, domCal.ID, imported, err.Error())
			return
		}
		imported++
		results = append(results, map[string]any{"event": mapEvent(stored), "existing": false})
	}

	if imported > 0 {
		publishEventsChanged(deps, domCal.ID)
	}
	Respond(w, req.ID, map[string]any{"calendarId": domCal.ID, "imported": imported, "events": results})
}

func respondImportError(w *ConnWriter, req Request, deps Deps, calendarID string, imported int, msg string) {
	if imported > 0 {
		publishEventsChanged(deps, calendarID)
	}
	RespondError(w, req.ID, msg)
}

func selectEvents(events []calendar.Event, uids []string) []calendar.Event {
	if len(uids) == 0 {
		return events
	}
	out := make([]calendar.Event, 0, len(uids))
	for _, ev := range events {
		if slices.Contains(uids, ev.UID) {
			out = append(out, ev)
		}
	}
	return out
}
