package ipc

import (
	"context"
	"fmt"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/taskconv"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func HandleTasks(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "tasks.list":
		handleTaskList(ctx, w, req, deps)
	case "tasks.get":
		handleTaskGet(ctx, w, req, deps)
	case "tasks.create":
		handleTaskCreate(ctx, w, req, deps)
	case "tasks.update":
		handleTaskUpdate(ctx, w, req, deps)
	case "tasks.complete":
		handleTaskComplete(ctx, w, req, deps)
	case "tasks.delete":
		handleTaskDelete(ctx, w, req, deps)
	default:
		RespondError(w, req.ID, "unknown tasks method: "+req.Method)
	}
}

func handleTaskList(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	filter := repo.TaskFilter{
		Query:            ParamString(req.Params, "query"),
		IncludeCompleted: true,
	}
	if _, ok := req.Params["includeCompleted"]; ok {
		filter.IncludeCompleted = ParamBool(req.Params, "includeCompleted")
	}
	if id := ParamString(req.Params, "calendarId"); id != "" {
		filter.CalendarIDs = []string{id}
	}

	tasks, total, err := deps.Repo.ListTasks(ctx, repo.ListTasksParams{
		Filter: filter,
		Limit:  ParamInt(req.Params, "limit"),
		Offset: ParamInt(req.Params, "offset"),
	})
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	Respond(w, req.ID, map[string]any{"tasks": mapTasks(tasks), "total": total})
}

func handleTaskGet(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	id := ParamString(req.Params, "id")
	if id == "" {
		RespondError(w, req.ID, "id is required")
		return
	}
	t, err := deps.Repo.GetTask(ctx, id)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	Respond(w, req.ID, mapTask(t))
}

func handleTaskCreate(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	calendarID := ParamString(req.Params, "calendarId")
	if calendarID == "" {
		RespondError(w, req.ID, "calendarId is required")
		return
	}

	t, err := taskFromParams(calendar.Task{Status: calendar.TaskNeedsAction}, req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	if t.Summary == "" {
		RespondError(w, req.ID, "summary is required")
		return
	}

	provider, writer, domCal, err := taskWriterForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	created, err := writer.CreateTask(ctx, domCal, &t)
	if err != nil {
		RespondError(w, req.ID, fmt.Sprintf("create task: %v", err))
		return
	}
	respondPersistedTask(ctx, w, req, deps, domCal.ID, created)
}

func handleTaskUpdate(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	entT, calendarID, ok := loadTaskCalendar(ctx, w, req, deps)
	if !ok {
		return
	}

	t, err := taskFromParams(taskconv.FromEnt(entT), req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	updateTask(ctx, w, req, deps, calendarID, &t)
}

func handleTaskComplete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	entT, calendarID, ok := loadTaskCalendar(ctx, w, req, deps)
	if !ok {
		return
	}

	t := taskconv.FromEnt(entT)
	done := true
	if _, ok := req.Params["completed"]; ok {
		done = ParamBool(req.Params, "completed")
	}
	switch {
	case done:
		t.Status = calendar.TaskCompleted
		t.PercentComplete = 100
		t.Completed = time.Now().UTC()
	default:
		t.Status = calendar.TaskNeedsAction
		t.PercentComplete = 0
		t.Completed = time.Time{}
	}
	updateTask(ctx, w, req, deps, calendarID, &t)
}

func handleTaskDelete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	entT, calendarID, ok := loadTaskCalendar(ctx, w, req, deps)
	if !ok {
		return
	}

	provider, writer, domCal, err := taskWriterForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	if err := writer.DeleteTask(ctx, domCal, taskconv.FromEnt(entT)); err != nil {
		RespondError(w, req.ID, fmt.Sprintf("delete task: %v", err))
		return
	}
	if err := deps.Repo.DeleteTask(ctx, entT.ID); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishTasksChanged(deps, calendarID)
	Respond(w, req.ID, map[string]any{"deleted": true})
}

// updateTask runs a domain task through its provider's writer and persists the
// result, the shared tail of tasks.update and tasks.complete.
func updateTask(ctx context.Context, w *ConnWriter, req Request, deps Deps, calendarID string, t *calendar.Task) {
	provider, writer, domCal, err := taskWriterForCalendar(ctx, deps, calendarID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	defer provider.Close()

	updated, err := writer.UpdateTask(ctx, domCal, t)
	if err != nil {
		RespondError(w, req.ID, fmt.Sprintf("update task: %v", err))
		return
	}
	respondPersistedTask(ctx, w, req, deps, domCal.ID, updated)
}

func respondPersistedTask(ctx context.Context, w *ConnWriter, req Request, deps Deps, calendarID string, t *calendar.Task) {
	stored, err := deps.Repo.UpsertTask(ctx, taskconv.UpsertInput(calendarID, t))
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}
	publishTasksChanged(deps, calendarID)
	Respond(w, req.ID, mapTask(stored))
}

// loadTaskCalendar resolves the task and its calendar id, responding with an
// error and reporting false when either is missing.
func loadTaskCalendar(ctx context.Context, w *ConnWriter, req Request, deps Deps) (*ent.Task, string, bool) {
	id := ParamString(req.Params, "id")
	if id == "" {
		RespondError(w, req.ID, "id is required")
		return nil, "", false
	}
	entT, err := deps.Repo.GetTask(ctx, id)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return nil, "", false
	}
	if entT.Edges.Calendar == nil {
		RespondError(w, req.ID, "task has no calendar")
		return nil, "", false
	}
	return entT, entT.Edges.Calendar.ID, true
}

func taskWriterForCalendar(ctx context.Context, deps Deps, calendarID string) (calendar.Provider, calendar.TaskWriter, calendar.Calendar, error) {
	provider, domCal, err := providerForCalendar(ctx, deps, calendarID)
	if err != nil {
		return nil, nil, calendar.Calendar{}, err
	}
	writer, ok := provider.(calendar.TaskWriter)
	if !ok {
		provider.Close()
		return nil, nil, calendar.Calendar{}, fmt.Errorf("account %q does not support tasks", domCal.AccountID)
	}
	return provider, writer, domCal, nil
}

// taskFromParams overlays request params onto base; only provided keys win.
func taskFromParams(base calendar.Task, p map[string]any) (calendar.Task, error) {
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
	if _, ok := p["parentUid"]; ok {
		base.ParentUID = ParamString(p, "parentUid")
	}
	if _, ok := p["priority"]; ok {
		base.Priority = ParamInt(p, "priority")
	}
	if _, ok := p["percentComplete"]; ok {
		base.PercentComplete = ParamInt(p, "percentComplete")
	}
	if _, ok := p["status"]; ok {
		base.Status = taskStatusFromParam(ParamString(p, "status"))
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

	if raw, ok := p["due"]; ok {
		when, err := parseOptionalTime(raw)
		if err != nil {
			return base, fmt.Errorf("due must be RFC3339: %v", err)
		}
		base.Due = when
	}
	if raw, ok := p["start"]; ok {
		when, err := parseOptionalTime(raw)
		if err != nil {
			return base, fmt.Errorf("start must be RFC3339: %v", err)
		}
		base.Start = when
	}
	return base, nil
}

// parseOptionalTime reads an RFC3339 time, treating an explicit empty string as
// a cleared value so an edit can drop a due date.
func parseOptionalTime(raw any) (time.Time, error) {
	s, _ := raw.(string)
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// recurrenceFromParam replaces the RRULE while preserving any provider RDATE/
// EXDATE. An empty rule clears recurrence (nil once nothing else remains).
func recurrenceFromParam(base *calendar.Recurrence, raw any) *calendar.Recurrence {
	var rrule []string
	switch v := raw.(type) {
	case string:
		if v != "" {
			rrule = []string{v}
		}
	case []string:
		rrule = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				rrule = append(rrule, s)
			}
		}
	}

	var rdate, exdate []string
	if base != nil {
		rdate, exdate = base.RDate, base.ExDate
	}
	if len(rrule)+len(rdate)+len(exdate) == 0 {
		return nil
	}
	return &calendar.Recurrence{RRule: rrule, RDate: rdate, ExDate: exdate}
}

func taskStatusFromParam(s string) calendar.TaskStatus {
	switch s {
	case "in_process":
		return calendar.TaskInProcess
	case "completed":
		return calendar.TaskCompleted
	case "cancelled":
		return calendar.TaskCancelled
	default:
		return calendar.TaskNeedsAction
	}
}

func publishTasksChanged(deps Deps, calendarID string) {
	if deps.Bus == nil {
		return
	}
	deps.Bus.Publish("tasks", map[string]any{"type": "changed", "calendarId": calendarID})
}

func mapTasks(items []*ent.Task) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, t := range items {
		out = append(out, mapTask(t))
	}
	return out
}

func mapTask(t *ent.Task) map[string]any {
	entry := map[string]any{
		"id":              t.ID,
		"uid":             t.UID,
		"summary":         t.Summary,
		"description":     t.Description,
		"location":        t.Location,
		"status":          string(t.Status),
		"priority":        t.Priority,
		"percentComplete": t.PercentComplete,
		"allDay":          t.AllDay,
		"parentUid":       t.ParentUID,
	}
	if cal := t.Edges.Calendar; cal != nil {
		entry["calendarId"] = cal.ID
	}
	if t.Due != nil {
		entry["due"] = *t.Due
	}
	if t.Start != nil {
		entry["start"] = *t.Start
	}
	if t.Completed != nil {
		entry["completed"] = *t.Completed
	}
	if len(t.Reminders) > 0 {
		entry["reminders"] = t.Reminders
	}
	if rr := calendar.RecurrenceFromMap(t.Recurrence); rr != nil && len(rr.RRule) > 0 {
		entry["recurrence"] = rr.RRule
	}
	return entry
}
