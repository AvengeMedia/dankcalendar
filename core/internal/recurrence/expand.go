package recurrence

import (
	"errors"
	"strings"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/tzcache"
	"github.com/teambition/rrule-go"
)

type Series struct {
	Start    time.Time
	AllDay   bool
	TimeZone string
	RRule    []string
	RDate    []string
	ExDate   []string
}

var ErrNoRule = errors.New("recurrence: series has no rrule or rdate")

// Expand returns occurrence start times within [from, to], inclusive.
func Expand(s Series, from, to time.Time) ([]time.Time, error) {
	if len(s.RRule) == 0 && len(s.RDate) == 0 {
		return nil, ErrNoRule
	}

	loc := s.location()
	dtstart := s.Start.In(loc)

	set := rrule.Set{}
	set.DTStart(dtstart)

	ruleCount := 0
	for _, line := range s.RRule {
		opt, err := rrule.StrToROptionInLocation(line, loc)
		if err != nil {
			continue
		}
		opt.Dtstart = dtstart
		rule, err := rrule.NewRRule(*opt)
		if err != nil {
			continue
		}
		set.RRule(rule)
		ruleCount++
	}

	for _, t := range parseDates(s.RDate, loc) {
		set.RDate(t)
		ruleCount++
	}
	if ruleCount == 0 {
		return nil, ErrNoRule
	}

	for _, t := range parseDates(s.ExDate, loc) {
		set.ExDate(t)
	}

	return set.Between(from, to, true), nil
}

func (s Series) location() *time.Location {
	switch {
	case s.AllDay:
		return time.UTC
	case s.TimeZone != "":
		if loc, err := tzcache.Load(s.TimeZone); err == nil {
			return loc
		}
	}
	return s.Start.Location()
}

// parseDates handles RDATE/EXDATE property values: comma-separated lists of
// date, local date-time, or UTC date-time forms.
func parseDates(values []string, loc *time.Location) []time.Time {
	var out []time.Time
	for _, value := range values {
		for raw := range strings.SplitSeq(value, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			t, err := ParseDateTime(raw, loc)
			if err != nil {
				continue
			}
			out = append(out, t)
		}
	}
	return out
}

// ParseDateTime parses an iCalendar date or date-time property value,
// interpreting floating local times in loc.
func ParseDateTime(raw string, loc *time.Location) (time.Time, error) {
	switch {
	case strings.HasSuffix(raw, "Z"):
		return time.ParseInLocation("20060102T150405Z", raw, time.UTC)
	case len(raw) == 8:
		return time.ParseInLocation("20060102", raw, loc)
	default:
		return time.ParseInLocation("20060102T150405", raw, loc)
	}
}
