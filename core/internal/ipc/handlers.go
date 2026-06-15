package ipc

import (
	"context"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func HandleCalendars(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "calendars.list":
		calendars, err := deps.Repo.ListCalendars(ctx)
		if err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, mapCalendars(calendars))
	case "calendars.setHidden":
		handleCalendarSetHidden(ctx, w, req, deps)
	case "calendars.rename":
		handleCalendarRename(ctx, w, req, deps)
	case "calendars.delete":
		handleCalendarDelete(ctx, w, req, deps)
	default:
		RespondError(w, req.ID, "unknown calendars method: "+req.Method)
	}
}

func HandleEvents(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "events.list":
		filter := repo.EventFilter{
			Query:            ParamString(req.Params, "query"),
			IncludeRecurring: true,
		}
		if from := ParamString(req.Params, "from"); from != "" {
			if t, err := time.Parse(time.RFC3339, from); err == nil {
				filter.From = &t
			}
		}
		if to := ParamString(req.Params, "to"); to != "" {
			if t, err := time.Parse(time.RFC3339, to); err == nil {
				filter.To = &t
			}
		}

		events, total, err := deps.Repo.ListEvents(ctx, repo.ListEventsParams{
			Filter: filter,
			Limit:  ParamInt(req.Params, "limit"),
			Offset: ParamInt(req.Params, "offset"),
		})
		if err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"events": mapEvents(events), "total": total})
	case "events.create":
		handleEventCreate(ctx, w, req, deps)
	case "events.update":
		handleEventUpdate(ctx, w, req, deps)
	case "events.delete":
		handleEventDelete(ctx, w, req, deps)
	default:
		RespondError(w, req.ID, "unknown events method: "+req.Method)
	}
}

func HandleSubscribe(w *ConnWriter, req Request, sub *Subscriber) {
	topics := ParamStringSlice(req.Params, "topics")
	if len(topics) == 0 {
		topics = []string{"accounts", "calendars", "events", "sync"}
	}
	sub.Subscribe(topics...)
	Respond(w, req.ID, map[string]any{"topics": sub.Topics()})
}

func HandleUnsubscribe(w *ConnWriter, req Request, sub *Subscriber) {
	topics := ParamStringSlice(req.Params, "topics")
	if len(topics) == 0 {
		Respond(w, req.ID, map[string]any{"topics": sub.Topics()})
		return
	}
	sub.Unsubscribe(topics...)
	Respond(w, req.ID, map[string]any{"topics": sub.Topics()})
}

func mapAccounts(items []*ent.Account) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		out = append(out, mapAccount(a))
	}
	return out
}

func mapAccount(a *ent.Account) map[string]any {
	return map[string]any{
		"id":          a.ID,
		"kind":        string(a.Kind),
		"displayName": a.DisplayName,
		"settings":    a.Settings,
		"needsReauth": a.NeedsReauth,
		"authError":   a.AuthError,
		"createdAt":   a.CreatedAt,
		"updatedAt":   a.UpdatedAt,
	}
}

func mapCalendars(items []*ent.Calendar) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, c := range items {
		name := c.Name
		if c.NameOverride != "" {
			name = c.NameOverride
		}
		entry := map[string]any{
			"id":           c.ID,
			"remoteId":     c.RemoteID,
			"name":         name,
			"providerName": c.Name,
			"description":  c.Description,
			"color":        c.Color,
			"timeZone":     c.TimeZone,
			"readOnly":     c.ReadOnly,
			"hidden":       c.Hidden,
			"updatedAt":    c.UpdatedAt,
		}
		if acc := c.Edges.Account; acc != nil {
			entry["accountId"] = acc.ID
			entry["accountKind"] = string(acc.Kind)
			entry["accountName"] = acc.DisplayName
		}
		out = append(out, entry)
	}
	return out
}

func mapEvents(items []*ent.Event) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, e := range items {
		out = append(out, mapEvent(e))
	}
	return out
}

func mapEvent(e *ent.Event) map[string]any {
	entry := map[string]any{
		"id":          e.ID,
		"uid":         e.UID,
		"summary":     e.Summary,
		"description": e.Description,
		"location":    e.Location,
		"url":         e.URL,
		"start":       e.Start,
		"end":         e.End,
		"allDay":      e.AllDay,
		"status":      string(e.Status),
		"recurringId": e.RecurringID,
	}
	if cal := e.Edges.Calendar; cal != nil {
		entry["calendarId"] = cal.ID
	}
	if len(e.Attendees) > 0 {
		entry["attendees"] = e.Attendees
	}
	if len(e.Organizer) > 0 {
		entry["organizer"] = e.Organizer
	}
	if len(e.Recurrence) > 0 {
		entry["recurrence"] = e.Recurrence
	}
	if len(e.Reminders) > 0 {
		entry["reminders"] = e.Reminders
	}
	return entry
}

func toEntEventStatus(s calendar.EventStatus) event.Status {
	switch s {
	case calendar.EventTentative:
		return event.StatusTentative
	case calendar.EventCancelled:
		return event.StatusCancelled
	}
	return event.StatusConfirmed
}
