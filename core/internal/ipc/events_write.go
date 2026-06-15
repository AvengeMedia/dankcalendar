package ipc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func handleEventCreate(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	calendarID := ParamString(req.Params, "calendarId")
	if calendarID == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}

	ev, err := eventFromParams(calendar.Event{Status: calendar.EventConfirmed}, req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	switch {
	case ev.Summary == "":
		RespondError(w, req.ID, "summary is required")
		return
	case ev.Start.IsZero() || ev.End.IsZero():
		RespondError(w, req.ID, "start and end are required (RFC3339)")
		return
	}

	provider, domCal, err := providerForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	created, err := provider.CreateEvent(ctx, domCal, &ev)
	if err != nil {
		RespondError(w, req.ID, fmt.Sprintf("create event: %v", err))
		return
	}

	stored, err := persistEvent(ctx, deps, domCal.ID, created)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishEventsChanged(deps, domCal.ID)
	Respond(w, req.ID, mapEvent(stored))
}

func handleEventUpdate(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "id")
	if id == "" {
		RespondError(w, req.ID, "id is required")
		return
	}

	entEv, err := deps.Repo.GetEvent(ctx, id)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	if entEv.Edges.Calendar == nil {
		RespondError(w, req.ID, "event has no calendar")
		return
	}
	calendarID := entEv.Edges.Calendar.ID

	ev, err := eventFromParams(domainEventFromEnt(entEv), req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	provider, domCal, err := providerForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	updated, err := provider.UpdateEvent(ctx, domCal, &ev)
	if err != nil {
		RespondError(w, req.ID, fmt.Sprintf("update event: %v", err))
		return
	}

	stored, err := persistEvent(ctx, deps, domCal.ID, updated)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishEventsChanged(deps, domCal.ID)
	Respond(w, req.ID, mapEvent(stored))
}

func handleEventDelete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "id")
	if id == "" {
		RespondError(w, req.ID, "id is required")
		return
	}

	entEv, err := deps.Repo.GetEvent(ctx, id)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	if entEv.Edges.Calendar == nil {
		RespondError(w, req.ID, "event has no calendar")
		return
	}
	calendarID := entEv.Edges.Calendar.ID

	provider, domCal, err := providerForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	if err := provider.DeleteEvent(ctx, domCal, domainEventFromEnt(entEv)); err != nil {
		RespondError(w, req.ID, fmt.Sprintf("delete event: %v", err))
		return
	}
	if err := deps.Repo.DeleteEvent(ctx, id); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishEventsChanged(deps, calendarID)
	Respond(w, req.ID, map[string]any{"deleted": true})
}

func handleCalendarSetHidden(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "calendarId")
	if id == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}
	if _, ok := req.Params["hidden"]; !ok {
		RespondError(w, req.ID, "hidden is required")
		return
	}

	if err := deps.Repo.SetCalendarHidden(ctx, id, ParamBool(req.Params, "hidden")); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	if deps.Bus != nil {
		deps.Bus.Publish("calendars", map[string]any{"type": "changed", "calendarId": id})
	}
	Respond(w, req.ID, map[string]any{"calendarId": id, "hidden": ParamBool(req.Params, "hidden")})
}

func handleCalendarRename(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "calendarId")
	if id == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}

	// Empty name clears the override, falling back to the provider name.
	name := strings.TrimSpace(ParamString(req.Params, "name"))
	if err := deps.Repo.SetCalendarNameOverride(ctx, id, name); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	if deps.Bus != nil {
		deps.Bus.Publish("calendars", map[string]any{"type": "changed", "calendarId": id})
	}
	Respond(w, req.ID, map[string]any{"calendarId": id, "name": name})
}

func handleCalendarSetReminders(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "calendarId")
	if id == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}

	// A missing or empty overrides object clears all overrides, reverting
	// the calendar to the global reminder settings.
	override := reminderOverrideFromParam(req.Params["overrides"])
	if err := deps.Repo.SetCalendarReminders(ctx, id, override); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	if deps.Bus != nil {
		deps.Bus.Publish("calendars", map[string]any{"type": "changed", "calendarId": id})
	}
	Respond(w, req.ID, map[string]any{"calendarId": id})
}

// reminderOverrideFromParam reads the override object: only the keys present
// become overrides, everything else inherits the global value.
func reminderOverrideFromParam(raw any) *settings.ReminderOverride {
	o := &settings.ReminderOverride{}
	m, ok := raw.(map[string]any)
	if !ok {
		return o
	}
	if v, ok := m["enabled"].(bool); ok {
		o.Enabled = &v
	}
	if v, ok := m["persist"].(bool); ok {
		o.Persist = &v
	}
	if v, ok := m["allDay"].(bool); ok {
		o.AllDay = &v
	}
	if v, ok := m["allDayTime"].(string); ok {
		o.AllDayTime = &v
	}
	if v, ok := intFromParam(m["allDayDaysBefore"]); ok {
		o.AllDayDaysBefore = &v
	}
	if v, ok := intFromParam(m["defaultReminderMinutes"]); ok {
		o.DefaultReminderMinutes = &v
	}
	if v, ok := intFromParam(m["snoozeMinutes"]); ok {
		o.SnoozeMinutes = &v
	}
	return o
}

func intFromParam(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}

func handleCalendarDelete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "calendarId")
	if id == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}

	if err := deps.Repo.DeleteCalendar(ctx, id); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	if deps.Bus != nil {
		deps.Bus.Publish("calendars", map[string]any{"type": "deleted", "calendarId": id})
		deps.Bus.Publish("events", map[string]any{"type": "changed", "calendarId": id})
	}
	Respond(w, req.ID, map[string]any{"deleted": true})
}

func providerForCalendar(ctx context.Context, deps Deps, calendarID string) (calendar.Provider, calendar.Calendar, error) {
	entCal, err := deps.Repo.GetCalendar(ctx, calendarID)
	if err != nil {
		return nil, calendar.Calendar{}, err
	}
	entAcc := entCal.Edges.Account
	if entAcc == nil {
		return nil, calendar.Calendar{}, errors.New("calendar has no account")
	}
	if entCal.ReadOnly {
		return nil, calendar.Calendar{}, fmt.Errorf("calendar %q is read-only", entCal.Name)
	}

	domAcc := calendar.Account{
		ID:          entAcc.ID,
		Kind:        calendar.AccountKind(entAcc.Kind),
		DisplayName: entAcc.DisplayName,
		Settings:    entAcc.Settings,
	}
	provider, err := deps.Registry.Build(ctx, domAcc, deps.Secrets)
	if err != nil {
		return nil, calendar.Calendar{}, err
	}

	domCal := calendar.Calendar{
		ID:        entCal.ID,
		AccountID: entAcc.ID,
		RemoteID:  entCal.RemoteID,
		Name:      entCal.Name,
		TimeZone:  entCal.TimeZone,
		ReadOnly:  entCal.ReadOnly,
	}
	return provider, domCal, nil
}

// eventFromParams overlays request params onto base; only provided keys win.
func eventFromParams(base calendar.Event, p map[string]any) (calendar.Event, error) {
	if _, ok := p["summary"]; ok {
		base.Summary = ParamString(p, "summary")
	}
	if _, ok := p["description"]; ok {
		base.Description = ParamString(p, "description")
	}
	if _, ok := p["location"]; ok {
		base.Location = ParamString(p, "location")
	}
	if _, ok := p["allDay"]; ok {
		base.AllDay = ParamBool(p, "allDay")
	}
	if _, ok := p["status"]; ok {
		switch ParamString(p, "status") {
		case "tentative":
			base.Status = calendar.EventTentative
		case "cancelled":
			base.Status = calendar.EventCancelled
		default:
			base.Status = calendar.EventConfirmed
		}
	}

	if raw, ok := p["reminders"]; ok {
		rems, err := remindersFromParam(raw)
		if err != nil {
			return base, err
		}
		base.Reminders = rems
	}

	for key, dst := range map[string]*time.Time{"start": &base.Start, "end": &base.End} {
		raw := ParamString(p, key)
		if raw == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return base, fmt.Errorf("%s must be RFC3339: %v", key, err)
		}
		*dst = t
	}

	if base.End.Before(base.Start) {
		return base, errors.New("end must not be before start")
	}
	return base, nil
}

// remindersFromParam parses the wire form: an array of {method?, minutes}
// objects. An empty array clears the event's reminders.
func remindersFromParam(raw any) ([]calendar.Reminder, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case []any:
		out := make([]calendar.Reminder, 0, len(v))
		for _, item := range v {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, errors.New("reminders entries must be objects")
			}
			minutes, ok := reminderMinutes(entry["minutes"])
			if !ok {
				return nil, errors.New("reminder minutes must be a number")
			}
			method, _ := entry["method"].(string)
			if method == "" {
				method = "popup"
			}
			out = append(out, calendar.Reminder{Method: method, Minutes: minutes})
		}
		return out, nil
	default:
		return nil, errors.New("reminders must be an array")
	}
}

func reminderMinutes(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func domainEventFromEnt(e *ent.Event) calendar.Event {
	return calendar.Event{
		ID:            e.ID,
		UID:           e.UID,
		RemoteID:      e.RemoteID,
		Etag:          e.Etag,
		Summary:       e.Summary,
		Description:   e.Description,
		Location:      e.Location,
		URL:           e.URL,
		Status:        calendar.EventStatus(e.Status),
		Start:         e.Start,
		End:           e.End,
		AllDay:        e.AllDay,
		StartTimeZone: e.StartTz,
		EndTimeZone:   e.EndTz,
		RecurringID:   e.RecurringID,
		Reminders:     calendar.RemindersFromMaps(e.Reminders),
	}
}

func persistEvent(ctx context.Context, deps Deps, calendarID string, ev *calendar.Event) (*ent.Event, error) {
	var organizer map[string]any
	if ev.Organizer != nil {
		organizer = ev.Organizer.ToMap()
	}
	return deps.Repo.UpsertEvent(ctx, repo.UpsertEventInput{
		CalendarID:    calendarID,
		UID:           ev.UID,
		RemoteID:      ev.RemoteID,
		Etag:          ev.Etag,
		Summary:       ev.Summary,
		Description:   ev.Description,
		Location:      ev.Location,
		URL:           ev.URL,
		Status:        toEntEventStatus(ev.Status),
		Start:         ev.Start,
		End:           ev.End,
		AllDay:        ev.AllDay,
		StartTZ:       ev.StartTimeZone,
		EndTZ:         ev.EndTimeZone,
		Recurrence:    ev.Recurrence.ToMap(),
		RecurringID:   ev.RecurringID,
		OriginalStart: ev.OriginalStart,
		Organizer:     organizer,
		Attendees:     calendar.AttendeesToMaps(ev.Attendees),
		Reminders:     calendar.RemindersToMaps(ev.Reminders),
		Categories:    ev.Categories,
		Transparency:  ev.Transparency,
		Visibility:    ev.Visibility,
		RawICS:        ev.RawICS,
	})
}

func publishEventsChanged(deps Deps, calendarID string) {
	if deps.Bus == nil {
		return
	}
	deps.Bus.Publish("events", map[string]any{"type": "changed", "calendarId": calendarID})
}
