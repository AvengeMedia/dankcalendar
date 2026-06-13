package local

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	ics "github.com/arran4/golang-ical"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

var durationPattern = regexp.MustCompile(`^([+-])?P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

func applyAlarms(ev *ics.VEvent, out *calendar.Event) {
	for _, alarm := range ev.Alarms() {
		switch strings.ToUpper(alarmProp(alarm, ics.ComponentPropertyAction)) {
		case "", string(ics.ActionDisplay), string(ics.ActionAudio):
		default:
			continue
		}

		minutes, ok := triggerMinutes(alarmProp(alarm, ics.ComponentPropertyTrigger))
		if !ok {
			continue
		}
		out.Reminders = append(out.Reminders, calendar.Reminder{Method: "popup", Minutes: minutes})
	}
}

func addAlarms(out *ics.VEvent, reminders []calendar.Reminder) {
	for _, rem := range reminders {
		if strings.EqualFold(rem.Method, "email") {
			continue
		}
		alarm := out.AddAlarm()
		alarm.SetAction(ics.ActionDisplay)
		alarm.SetTrigger(triggerFromMinutes(rem.Minutes))
	}
}

func alarmProp(alarm *ics.VAlarm, key ics.ComponentProperty) string {
	prop := alarm.GetProperty(key)
	if prop == nil {
		return ""
	}
	return prop.Value
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
