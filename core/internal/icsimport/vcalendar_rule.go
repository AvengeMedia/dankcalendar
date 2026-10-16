package icsimport

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/teambition/rrule-go"
)

func legacyRule(raw string) (string, error) {
	if strings.Contains(raw, "FREQ=") {
		return raw, nil
	}
	fields := strings.Fields(strings.ToUpper(raw))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty vCalendar recurrence")
	}
	var prefix, frequency, by string
	for _, spec := range []struct{ prefix, frequency, by string }{
		{"MP", "MONTHLY", "BYDAY"}, {"MD", "MONTHLY", "BYMONTHDAY"},
		{"YM", "YEARLY", "BYMONTH"}, {"YD", "YEARLY", "BYYEARDAY"},
		{"D", "DAILY", ""}, {"W", "WEEKLY", "BYDAY"},
	} {
		if strings.HasPrefix(fields[0], spec.prefix) {
			prefix, frequency, by = spec.prefix, spec.frequency, spec.by
			break
		}
	}
	interval, err := strconv.Atoi(strings.TrimPrefix(fields[0], prefix))
	if prefix == "" || err != nil || interval < 1 {
		return "", fmt.Errorf("unsupported vCalendar recurrence %q", raw)
	}
	parts := []string{"FREQ=" + frequency, "INTERVAL=" + strconv.Itoa(interval)}
	var modifiers []string
	ordinal := ""
	for _, field := range fields[1:] {
		switch {
		case strings.HasPrefix(field, "#"):
			count, err := strconv.Atoi(field[1:])
			if err != nil || count < 0 {
				return "", fmt.Errorf("invalid vCalendar recurrence duration")
			}
			// A duration counts frequency periods, not individual weekdays. Reject
			// multi-day bounded rules until they can be represented without changing dates.
			if count > 0 {
				if len(modifiers) > 1 {
					return "", fmt.Errorf("bounded multi-day vCalendar recurrence requires an iCalendar 2.0 export")
				}
				parts = append(parts, "COUNT="+strconv.Itoa(count))
			}
		case strings.Contains(field, "T") && len(field) >= 15:
			parts = append(parts, "UNTIL="+field)
		case prefix == "MP" && (strings.HasSuffix(field, "+") || strings.HasSuffix(field, "-")):
			ordinal = signedOrdinal(field)
		default:
			if by == "" {
				return "", fmt.Errorf("unsupported vCalendar recurrence modifier %q", field)
			}
			switch prefix {
			case "MP":
				field = ordinal + field
			case "MD":
				field = signedOrdinal(field)
			}
			modifiers = append(modifiers, field)
		}
	}
	if len(modifiers) > 0 {
		parts = append(parts, by+"="+strings.Join(modifiers, ","))
	}
	rule := strings.Join(parts, ";")
	if _, err := rrule.StrToROption(rule); err != nil {
		return "", fmt.Errorf("invalid vCalendar recurrence: %w", err)
	}
	return rule, nil
}

func signedOrdinal(value string) string {
	switch {
	case value == "LD":
		return "-1"
	case strings.HasSuffix(value, "-"):
		return "-" + strings.TrimSuffix(value, "-")
	default:
		return strings.TrimSuffix(value, "+")
	}
}
