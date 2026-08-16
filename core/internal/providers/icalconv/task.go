package icalconv

import (
	"strconv"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

// CalendarFromTask wraps a single task in its own VCALENDAR, as written to a
// CalDAV object or a one-task file.
func CalendarFromTask(t *cal.Task, uid string) *ical.Calendar {
	out := NewCalendar()
	out.Children = append(out.Children, BuildTask(t, uid))
	return out
}

// TaskFromComponent maps a VTODO to a domain task. It reports false when the
// component has no UID. RemoteID and Etag are left to the caller, which knows
// the transport (a file UID or a CalDAV path/etag).
func TaskFromComponent(calID string, comp *ical.Component, tz *TZResolver) (cal.Task, bool) {
	uid := ComponentUID(comp)
	if uid == "" {
		return cal.Task{}, false
	}

	t := cal.Task{
		CalendarID:      calID,
		UID:             uid,
		Summary:         propText(comp, ical.PropSummary),
		Description:     propText(comp, ical.PropDescription),
		Location:        propText(comp, ical.PropLocation),
		Status:          taskStatusFromComponent(comp),
		Priority:        intProp(comp, ical.PropPriority),
		PercentComplete: intProp(comp, ical.PropPercentComplete),
		ParentUID:       parentUID(comp),
		Recurrence:      recurrenceFromComponent(comp, tz),
		Reminders:       remindersFromComponent(comp),
	}

	if due, zone, allDay, ok := parseDateTime(comp.Props.Get(ical.PropDue), tz); ok {
		t.Due = due
		t.DueTimeZone = zone
		t.AllDay = allDay
	}
	if start, zone, allDay, ok := parseDateTime(comp.Props.Get(ical.PropDateTimeStart), tz); ok {
		t.Start = start
		t.StartTimeZone = zone
		if t.Due.IsZero() {
			t.AllDay = allDay
		}
	}
	if done, _, _, ok := parseDateTime(comp.Props.Get(ical.PropCompleted), tz); ok {
		t.Completed = done
	}

	return t, true
}

// BuildTask renders a domain task as a VTODO component.
func BuildTask(t *cal.Task, uid string) *ical.Component {
	comp := ical.NewComponent(ical.CompToDo)
	props := comp.Props
	props.SetText(ical.PropUID, uid)
	props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	props.SetText(ical.PropSummary, t.Summary)
	if t.Description != "" {
		props.SetText(ical.PropDescription, t.Description)
	}
	if t.Location != "" {
		props.SetText(ical.PropLocation, t.Location)
	}
	if status := taskStatusValue(t.Status); status != "" {
		props.SetText(ical.PropStatus, status)
	}
	if t.Priority > 0 {
		setRaw(props, ical.PropPriority, strconv.Itoa(t.Priority))
	}
	if t.PercentComplete > 0 {
		setRaw(props, ical.PropPercentComplete, strconv.Itoa(t.PercentComplete))
	}
	if t.ParentUID != "" {
		setRaw(props, ical.PropRelatedTo, t.ParentUID)
	}

	setTaskDate(props, ical.PropDue, t.Due, t.DueTimeZone, t.AllDay)
	setTaskDate(props, ical.PropDateTimeStart, t.Start, t.StartTimeZone, t.AllDay)
	if !t.Completed.IsZero() {
		props.SetDateTime(ical.PropCompleted, t.Completed.UTC())
	}

	setRecurrence(props, t.Recurrence)
	addAlarms(comp, t.Reminders)

	return comp
}

func setTaskDate(props ical.Props, name string, when time.Time, tz string, allDay bool) {
	switch {
	case when.IsZero():
		return
	case allDay:
		props.SetDate(name, when)
	default:
		props.SetDateTime(name, inZone(when, tz))
	}
}

func taskStatusFromComponent(comp *ical.Component) cal.TaskStatus {
	prop := comp.Props.Get(ical.PropStatus)
	if prop == nil {
		return cal.TaskNeedsAction
	}
	switch strings.ToUpper(prop.Value) {
	case "IN-PROCESS":
		return cal.TaskInProcess
	case "COMPLETED":
		return cal.TaskCompleted
	case "CANCELLED":
		return cal.TaskCancelled
	default:
		return cal.TaskNeedsAction
	}
}

func taskStatusValue(status cal.TaskStatus) string {
	switch status {
	case cal.TaskInProcess:
		return "IN-PROCESS"
	case cal.TaskCompleted:
		return "COMPLETED"
	case cal.TaskCancelled:
		return "CANCELLED"
	default:
		return "NEEDS-ACTION"
	}
}

func intProp(comp *ical.Component, name string) int {
	prop := comp.Props.Get(name)
	if prop == nil {
		return 0
	}
	n, err := prop.Int()
	if err != nil {
		return 0
	}
	return n
}

// parentUID returns the first RELATED-TO value, preferring an explicit
// RELTYPE=PARENT relation over an unqualified one.
func parentUID(comp *ical.Component) string {
	var fallback string
	for _, prop := range comp.Props.Values(ical.PropRelatedTo) {
		reltype := strings.ToUpper(prop.Params.Get("RELTYPE"))
		if reltype == "PARENT" {
			return prop.Value
		}
		if reltype == "" && fallback == "" {
			fallback = prop.Value
		}
	}
	return fallback
}
