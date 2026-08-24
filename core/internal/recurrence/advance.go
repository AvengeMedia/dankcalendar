package recurrence

import (
	"time"

	"github.com/teambition/rrule-go"
)

// Advance returns the occurrence after Start together with the series' rules
// rebased onto it, decrementing any COUNT bound. ok is false when the series
// has no further occurrences.
func Advance(s Series) (next time.Time, rules []string, ok bool) {
	if len(s.RRule) == 0 {
		return time.Time{}, nil, false
	}

	loc := s.location()
	dtstart := s.Start.In(loc)
	set := rrule.Set{}
	set.DTStart(dtstart)

	kept := make([]string, 0, len(s.RRule))
	for _, line := range s.RRule {
		opt, err := rrule.StrToROptionInLocation(line, loc)
		if err != nil {
			continue
		}
		stored := *opt
		opt.Dtstart = dtstart
		rule, err := rrule.NewRRule(*opt)
		if err != nil {
			continue
		}
		set.RRule(rule)
		if stored.Count > 0 {
			stored.Count--
		}
		kept = append(kept, stored.RRuleString())
	}
	if len(kept) == 0 {
		return time.Time{}, nil, false
	}

	for _, t := range parseDates(s.ExDate, loc) {
		set.ExDate(t)
	}

	next = set.After(dtstart, false)
	if next.IsZero() {
		return time.Time{}, nil, false
	}
	return next, kept, true
}
