package caldav

import (
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const prodID = "-//dankcalendar//dankcalendar//EN"

func eventsFromObject(c cal.Calendar, obj caldav.CalendarObject) []cal.Event {
	if obj.Data == nil {
		return nil
	}

	var out []cal.Event
	for _, comp := range obj.Data.Events() {
		ev, ok := eventFromComponent(c, comp.Component)
		if !ok {
			continue
		}
		ev.RemoteID = obj.Path
		ev.Etag = obj.ETag
		out = append(out, ev)
	}
	return out
}

func eventFromComponent(c cal.Calendar, comp *ical.Component) (cal.Event, bool) {
	uidProp := comp.Props.Get(ical.PropUID)
	if uidProp == nil || uidProp.Value == "" {
		return cal.Event{}, false
	}
	uid := uidProp.Value

	ev := cal.Event{
		CalendarID:  c.ID,
		UID:         uid,
		Summary:     propText(comp, ical.PropSummary),
		Description: propText(comp, ical.PropDescription),
		Location:    propText(comp, ical.PropLocation),
		Status:      statusFromComponent(comp),
	}

	if rid := comp.Props.Get(ical.PropRecurrenceID); rid != nil {
		ev.UID = uid + "/" + rid.Value
		ev.RecurringID = uid
		if t, err := rid.DateTime(time.UTC); err == nil {
			ev.OriginalStart = t
		}
	}
	if urlProp := comp.Props.Get(ical.PropURL); urlProp != nil {
		ev.URL = urlProp.Value
	}

	applyTimes(comp, &ev)
	applyRecurrence(comp, &ev)
	applyAttendees(comp, &ev)

	return ev, true
}

func applyTimes(comp *ical.Component, ev *cal.Event) {
	startProp := comp.Props.Get(ical.PropDateTimeStart)
	if startProp != nil {
		if start, err := startProp.DateTime(time.UTC); err == nil {
			ev.Start = start
		}
		ev.AllDay = startProp.ValueType() == ical.ValueDate
	}

	endProp := comp.Props.Get(ical.PropDateTimeEnd)
	durProp := comp.Props.Get(ical.PropDuration)
	switch {
	case endProp != nil:
		if end, err := endProp.DateTime(time.UTC); err == nil {
			ev.End = end
			return
		}
		ev.End = ev.Start
	case durProp != nil:
		dur, err := durProp.Duration()
		if err != nil {
			ev.End = ev.Start
			return
		}
		ev.End = ev.Start.Add(dur)
	case ev.AllDay:
		ev.End = ev.Start.Add(24 * time.Hour)
	default:
		ev.End = ev.Start
	}
}

func applyRecurrence(comp *ical.Component, ev *cal.Event) {
	rrules := rawValues(comp, ical.PropRecurrenceRule)
	rdates := rawValues(comp, ical.PropRecurrenceDates)
	exdates := rawValues(comp, ical.PropExceptionDates)
	if len(rrules)+len(rdates)+len(exdates) == 0 {
		return
	}
	ev.Recurrence = &cal.Recurrence{RRule: rrules, RDate: rdates, ExDate: exdates}
}

func applyAttendees(comp *ical.Component, ev *cal.Event) {
	for _, prop := range comp.Props.Values(ical.PropAttendee) {
		ev.Attendees = append(ev.Attendees, attendeeFromProp(prop))
	}
	org := comp.Props.Get(ical.PropOrganizer)
	if org == nil {
		return
	}
	attendee := attendeeFromProp(*org)
	attendee.Organizer = true
	ev.Organizer = &attendee
}

func attendeeFromProp(prop ical.Prop) cal.Attendee {
	role := prop.Params.Get(ical.ParamRole)
	return cal.Attendee{
		Email:       stripMailto(prop.Value),
		DisplayName: prop.Params.Get(ical.ParamCommonName),
		Role:        role,
		Status:      strings.ToLower(prop.Params.Get(ical.ParamParticipationStatus)),
		Optional:    strings.EqualFold(role, "OPT-PARTICIPANT"),
	}
}

func statusFromComponent(comp *ical.Component) cal.EventStatus {
	prop := comp.Props.Get(ical.PropStatus)
	if prop == nil {
		return cal.EventConfirmed
	}
	switch strings.ToUpper(prop.Value) {
	case "TENTATIVE":
		return cal.EventTentative
	case "CANCELLED":
		return cal.EventCancelled
	default:
		return cal.EventConfirmed
	}
}

func propText(comp *ical.Component, name string) string {
	text, err := comp.Props.Text(name)
	if err != nil {
		return ""
	}
	return text
}

func rawValues(comp *ical.Component, name string) []string {
	props := comp.Props.Values(name)
	if len(props) == 0 {
		return nil
	}
	out := make([]string, 0, len(props))
	for _, prop := range props {
		out = append(out, prop.Value)
	}
	return out
}

func stripMailto(value string) string {
	if len(value) >= 7 && strings.EqualFold(value[:7], "mailto:") {
		return value[7:]
	}
	return value
}

func icalFromEvent(ev *cal.Event, uid string) *ical.Calendar {
	event := ical.NewEvent()
	props := event.Props
	props.SetText(ical.PropUID, uid)
	props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	props.SetText(ical.PropSummary, ev.Summary)
	if ev.Description != "" {
		props.SetText(ical.PropDescription, ev.Description)
	}
	if ev.Location != "" {
		props.SetText(ical.PropLocation, ev.Location)
	}
	if ev.URL != "" {
		setRaw(props, ical.PropURL, ev.URL)
	}
	if status := statusValue(ev.Status); status != "" {
		props.SetText(ical.PropStatus, status)
	}

	setEventTimes(props, ev)
	setRecurrence(props, ev.Recurrence)
	setAttendees(props, ev)

	out := ical.NewCalendar()
	out.Props.SetText(ical.PropVersion, "2.0")
	out.Props.SetText(ical.PropProductID, prodID)
	out.Children = append(out.Children, event.Component)
	return out
}

func setEventTimes(props ical.Props, ev *cal.Event) {
	switch {
	case ev.AllDay:
		props.SetDate(ical.PropDateTimeStart, ev.Start)
		end := ev.End
		if !end.After(ev.Start) {
			end = ev.Start.Add(24 * time.Hour)
		}
		props.SetDate(ical.PropDateTimeEnd, end)
	default:
		props.SetDateTime(ical.PropDateTimeStart, ev.Start.UTC())
		end := ev.End
		if end.IsZero() {
			end = ev.Start
		}
		props.SetDateTime(ical.PropDateTimeEnd, end.UTC())
	}
}

func setRecurrence(props ical.Props, rec *cal.Recurrence) {
	if rec == nil {
		return
	}
	for _, rule := range rec.RRule {
		setRawAdd(props, ical.PropRecurrenceRule, rule)
	}
	for _, rdate := range rec.RDate {
		setRawAdd(props, ical.PropRecurrenceDates, rdate)
	}
	for _, exdate := range rec.ExDate {
		setRawAdd(props, ical.PropExceptionDates, exdate)
	}
}

func setAttendees(props ical.Props, ev *cal.Event) {
	for _, attendee := range ev.Attendees {
		props.Add(attendeeProp(ical.PropAttendee, attendee))
	}
	if ev.Organizer == nil || ev.Organizer.Email == "" {
		return
	}
	props.Set(attendeeProp(ical.PropOrganizer, *ev.Organizer))
}

func attendeeProp(name string, attendee cal.Attendee) *ical.Prop {
	prop := ical.NewProp(name)
	prop.Value = "mailto:" + attendee.Email
	if attendee.DisplayName != "" {
		prop.Params.Set(ical.ParamCommonName, attendee.DisplayName)
	}
	if attendee.Role != "" {
		prop.Params.Set(ical.ParamRole, attendee.Role)
	}
	if attendee.Status != "" {
		prop.Params.Set(ical.ParamParticipationStatus, strings.ToUpper(attendee.Status))
	}
	return prop
}

func statusValue(status cal.EventStatus) string {
	switch status {
	case cal.EventConfirmed:
		return "CONFIRMED"
	case cal.EventTentative:
		return "TENTATIVE"
	case cal.EventCancelled:
		return "CANCELLED"
	}
	return ""
}

func setRaw(props ical.Props, name, value string) {
	prop := ical.NewProp(name)
	prop.Value = value
	props.Set(prop)
}

func setRawAdd(props ical.Props, name, value string) {
	prop := ical.NewProp(name)
	prop.Value = value
	props.Add(prop)
}
