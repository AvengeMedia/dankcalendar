package icalconv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	ical "github.com/emersion/go-ical"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

var durationPattern = regexp.MustCompile(`^([+-])?P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

func applyAlarms(comp *ical.Component, ev *cal.Event) {
	for _, alarm := range comp.Children {
		if alarm.Name != ical.CompAlarm {
			continue
		}
		switch strings.ToUpper(propValue(alarm, ical.PropAction)) {
		case "", "DISPLAY", "AUDIO":
		default:
			continue
		}
		minutes, ok := triggerMinutes(propValue(alarm, ical.PropTrigger))
		if !ok {
			continue
		}
		ev.Reminders = append(ev.Reminders, cal.Reminder{Method: "popup", Minutes: minutes})
	}
}

func addAlarms(event *ical.Event, reminders []cal.Reminder) {
	for _, rem := range reminders {
		if strings.EqualFold(rem.Method, "email") {
			continue
		}
		alarm := ical.NewComponent(ical.CompAlarm)
		alarm.Props.SetText(ical.PropAction, "DISPLAY")
		// DISPLAY alarms require a description per RFC 5545.
		alarm.Props.SetText(ical.PropDescription, "Reminder")
		setRaw(alarm.Props, ical.PropTrigger, triggerFromMinutes(rem.Minutes))
		event.Children = append(event.Children, alarm)
	}
}

func propValue(comp *ical.Component, name string) string {
	if p := comp.Props.Get(name); p != nil {
		return p.Value
	}
	return ""
}

// triggerMinutes converts a relative TRIGGER duration into minutes before the
// event start (positive = before, matching provider reminder semantics).
// Absolute date-time triggers are not supported.
func triggerMinutes(raw string) (int, bool) {
	match := durationPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return 0, false
	}

	total := durationField(match[2])*7*24*60 +
		durationField(match[3])*24*60 +
		durationField(match[4])*60 +
		durationField(match[5])
	if durationField(match[6]) >= 30 {
		total++
	}

	if match[1] == "-" {
		return total, true
	}
	return -total, true
}

func triggerFromMinutes(minutes int) string {
	switch {
	case minutes == 0:
		return "PT0S"
	case minutes > 0:
		return fmt.Sprintf("-PT%dM", minutes)
	default:
		return fmt.Sprintf("PT%dM", -minutes)
	}
}

func durationField(raw string) int {
	if raw == "" {
		return 0
	}
	n, _ := strconv.Atoi(raw)
	return n
}
