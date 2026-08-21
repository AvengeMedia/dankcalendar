package ipc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/eventconv"
	"github.com/AvengeMedia/dankcalendar/core/internal/rsvp"
	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
	"github.com/AvengeMedia/dankcalendar/core/internal/tzcache"
	"github.com/AvengeMedia/dankgo/log"
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

	// Locally created events carry the machine's zone so providers write a
	// TZID instead of forcing UTC (#80).
	if tz := tzcache.LocalName(); !ev.AllDay && tz != "" {
		if ev.StartTimeZone == "" {
			ev.StartTimeZone = tz
		}
		if ev.EndTimeZone == "" {
			ev.EndTimeZone = tz
		}
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

	if raw := ParamString(req.Params, "occurrenceStart"); raw != "" && len(entEv.Recurrence) > 0 {
		occStart, perr := parseOccurrenceStart(raw, entEv.AllDay)
		if perr != nil {
			RespondError(w, req.ID, perr.Error())
			return
		}
		ev = shiftSeriesTimes(ev, entEv.Start, occStart)
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

	if raw := ParamString(req.Params, "occurrenceStart"); raw != "" {
		occStart, perr := parseOccurrenceStart(raw, entEv.AllDay)
		if perr != nil {
			RespondError(w, req.ID, perr.Error())
			return
		}
		deleteEventOccurrence(ctx, w, req, deps, entEv, calendarID, occStart)
		return
	}

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

// deleteEventOccurrence removes one instance of a recurring series by adding
// an EXDATE to the master and updating it in place.
func deleteEventOccurrence(ctx context.Context, w *ConnWriter, req Request, deps Deps, entEv *ent.Event, calendarID string, occStart time.Time) {
	ev := domainEventFromEnt(entEv)
	if ev.Recurrence == nil || len(ev.Recurrence.RRule)+len(ev.Recurrence.RDate) == 0 {
		RespondError(w, req.ID, "event is not recurring")
		return
	}

	ev.Recurrence.ExDate = append(ev.Recurrence.ExDate, exDateValue(occStart, ev.AllDay))

	provider, domCal, err := providerForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	updated, err := provider.UpdateEvent(ctx, domCal, &ev)
	if err != nil {
		RespondError(w, req.ID, fmt.Sprintf("delete occurrence: %v", err))
		return
	}
	if _, err := persistEvent(ctx, deps, domCal.ID, updated); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishEventsChanged(deps, calendarID)
	Respond(w, req.ID, map[string]any{"deleted": true})
}

func handleEventRSVP(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "id")
	if id == "" {
		RespondError(w, req.ID, "id is required")
		return
	}
	response := ParamString(req.Params, "response")
	if response == "" {
		RespondError(w, req.ID, "response is required (accept|decline|tentative)")
		return
	}
	var occurrenceStart time.Time
	if raw := ParamString(req.Params, "occurrenceStart"); raw != "" {
		parsed, perr := time.Parse(time.RFC3339, raw)
		if perr != nil {
			RespondError(w, req.ID, fmt.Sprintf("occurrenceStart must be RFC3339: %v", perr))
			return
		}
		occurrenceStart = parsed
	}

	res, err := rsvp.Apply(ctx, rsvp.Stores{
		Repo:     deps.Repo,
		Registry: deps.Registry,
		Secrets:  deps.Secrets,
	}, id, response, occurrenceStart)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishEventsChanged(deps, res.CalendarID)

	stored, err := deps.Repo.GetEvent(ctx, res.EventID)
	if err != nil {
		Respond(w, req.ID, map[string]any{"id": res.EventID, "response": res.Response})
		return
	}
	Respond(w, req.ID, mapEvent(stored))
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

func handleCalendarSetSyncDisabled(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "calendarId")
	if id == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}
	if _, ok := req.Params["disabled"]; !ok {
		RespondError(w, req.ID, "disabled is required")
		return
	}

	cal, err := deps.Repo.GetCalendar(ctx, id)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	disabled := ParamBool(req.Params, "disabled")
	if err := deps.Repo.SetCalendarSyncDisabled(ctx, id, disabled); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	// Re-enabling resyncs in the background so the calendar's events come
	// back without waiting for the next scheduled pass; the response must
	// not block on a full account sync.
	if !disabled && deps.Sync != nil && cal.Edges.Account != nil {
		acc := cal.Edges.Account
		go func() {
			if err := deps.Sync.SyncAccount(context.Background(), acc); err != nil {
				log.Warnf("sync after calendar enable: %v", err)
			}
		}()
	}

	if deps.Bus != nil {
		deps.Bus.Publish("calendars", map[string]any{"type": "changed", "calendarId": id})
		deps.Bus.Publish("events", map[string]any{"type": "changed", "calendarId": id})
	}
	Respond(w, req.ID, map[string]any{"calendarId": id, "syncDisabled": disabled})
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
		ID:                  entCal.ID,
		AccountID:           entAcc.ID,
		RemoteID:            entCal.RemoteID,
		Name:                entCal.Name,
		TimeZone:            entCal.TimeZone,
		ReadOnly:            entCal.ReadOnly,
		SupportedComponents: entCal.SupportedComponents,
	}
	return provider, domCal, nil
}

// exDateValue formats an occurrence start as an EXDATE property value in the
// UTC forms the expander compares against.
func exDateValue(occStart time.Time, allDay bool) string {
	if allDay {
		return occStart.UTC().Format("20060102")
	}
	return occStart.UTC().Format("20060102T150405Z")
}

// allDayDate pins an all-day instant to UTC midnight of the calendar date it
// was written with, so a client's local offset never shifts the day.
func allDayDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func parseOccurrenceStart(raw string, allDay bool) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("occurrenceStart must be RFC3339: %v", err)
	}
	if allDay {
		return allDayDate(t), nil
	}
	return t, nil
}

func shiftSeriesTimes(ev calendar.Event, masterStart, occurrenceStart time.Time) calendar.Event {
	duration := ev.End.Sub(ev.Start)
	ev.Start = masterStart.Add(ev.Start.Sub(occurrenceStart))
	ev.End = ev.Start.Add(duration)
	return ev
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

	if raw, ok := p["recurrence"]; ok {
		base.Recurrence = recurrenceFromParam(base.Recurrence, raw)
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
		if base.AllDay {
			t = allDayDate(t)
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
	ev := calendar.Event{
		ID:            e.ID,
		UID:           e.UID,
		RemoteID:      e.RemoteID,
		Etag:          e.Etag,
		Summary:       e.Summary,
		Description:   e.Description,
		Location:      e.Location,
		URL:           e.URL,
		MeetingURL:    e.MeetingURL,
		Status:        calendar.EventStatus(e.Status),
		Start:         e.Start,
		End:           e.End,
		AllDay:        e.AllDay,
		StartTimeZone: e.StartTz,
		EndTimeZone:   e.EndTz,
		Recurrence:    calendar.RecurrenceFromMap(e.Recurrence),
		RecurringID:   e.RecurringID,
		Reminders:     calendar.RemindersFromMaps(e.Reminders),
	}
	if e.OriginalStart != nil {
		ev.OriginalStart = *e.OriginalStart
	}
	return ev
}

func persistEvent(ctx context.Context, deps Deps, calendarID string, ev *calendar.Event) (*ent.Event, error) {
	return deps.Repo.UpsertEvent(ctx, eventconv.UpsertInput(calendarID, ev))
}

func publishEventsChanged(deps Deps, calendarID string) {
	if deps.Bus == nil {
		return
	}
	deps.Bus.Publish("events", map[string]any{"type": "changed", "calendarId": calendarID})
}
