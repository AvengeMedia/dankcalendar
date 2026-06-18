package icalconv

import (
	"strconv"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
	"github.com/teambition/rrule-go"
)

// TZResolver maps iCalendar TZID labels to time.Location values. go-ical only
// resolves a TZID through time.LoadLocation, so non-IANA labels common in
// forwarded events (Windows names like "Eastern Standard Time", or Outlook's
// "Customized Time Zone") fail and leave the event with a zero time. The
// resolver fills that gap using, in order: the IANA database, a Windows-name
// fallback, and any VTIMEZONE block carried alongside the event.
type TZResolver struct {
	defs map[string]*tzDef
	// defaultLoc interprets floating (no TZID, no Z) values, per RFC 5545 §3.3.5:
	// the calendar's own zone when known, otherwise the viewer's local zone.
	defaultLoc  *time.Location
	defaultZone string
}

type tzDef struct {
	subs []tzSub
}

// tzSub is one STANDARD or DAYLIGHT block: the offset it switches to and the
// onset times at which it takes effect (floating wall-clock, compared by order).
type tzSub struct {
	offset int
	set    *rrule.Set
}

// NewTZResolver indexes the VTIMEZONE components of a calendar document and
// records the zone used for floating values: calTZ (the calendar's configured
// zone) when set, else the document's X-WR-TIMEZONE, else the local zone. A nil
// calendar still yields a usable resolver for IANA and Windows names.
func NewTZResolver(c *ical.Calendar, calTZ string) *TZResolver {
	r := &TZResolver{defs: map[string]*tzDef{}, defaultLoc: time.Local}
	if loc, name := loadZone(calTZ); loc != nil {
		r.defaultLoc, r.defaultZone = loc, name
	}
	if c == nil {
		return r
	}
	if r.defaultZone == "" {
		if p := c.Props.Get("X-WR-TIMEZONE"); p != nil {
			if loc, name := loadZone(p.Value); loc != nil {
				r.defaultLoc, r.defaultZone = loc, name
			}
		}
	}
	for _, child := range c.Children {
		if child.Name != ical.CompTimezone {
			continue
		}
		idProp := child.Props.Get(ical.PropTimezoneID)
		if idProp == nil || idProp.Value == "" {
			continue
		}
		if def := parseTZDef(child); def != nil {
			r.defs[idProp.Value] = def
		}
	}
	return r
}

// loadZone resolves a zone name through the IANA database, then the Windows-name
// fallback. It returns nil when the name is empty or unresolvable.
func loadZone(name string) (*time.Location, string) {
	if name == "" {
		return nil, ""
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc, name
	}
	if iana := windowsZones[name]; iana != "" {
		if loc, err := time.LoadLocation(iana); err == nil {
			return loc, iana
		}
	}
	return nil, ""
}

// floating returns the zone for floating values. The name is empty when it
// resolved to the local zone, which has no portable identifier.
func (r *TZResolver) floating() (*time.Location, string) {
	if r == nil || r.defaultLoc == nil {
		return time.UTC, ""
	}
	return r.defaultLoc, r.defaultZone
}

// resolveValue parses a single RDATE/EXDATE date-time value (TZID-qualified or
// floating) into an absolute instant, so exclusions match occurrences regardless
// of which zone the property used.
func (r *TZResolver) resolveValue(tzid, raw string) (time.Time, bool) {
	var loc *time.Location
	if tzid == "" {
		loc, _ = r.floating()
	} else {
		loc, _ = r.location(tzid, mustNaive(raw))
	}
	t, err := time.ParseInLocation("20060102T150405", raw, loc)
	return t, err == nil
}

func mustNaive(raw string) time.Time {
	t, _ := time.ParseInLocation("20060102T150405", raw, time.UTC)
	return t
}

// location resolves tzid for an event at wall-clock when. It reports the IANA
// name when one is known so recurrence expansion can reload it; a
// VTIMEZONE-derived zone has no name and yields a fixed offset that is correct
// for the instant but cannot carry DST transitions. An unresolvable tzid falls
// back to UTC, preserving the literal wall-clock instead of discarding the time.
func (r *TZResolver) location(tzid string, when time.Time) (*time.Location, string) {
	if tzid == "" {
		return time.UTC, ""
	}
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc, tzid
	}
	if iana := windowsZones[tzid]; iana != "" {
		if loc, err := time.LoadLocation(iana); err == nil {
			return loc, iana
		}
	}
	if r != nil {
		if def := r.defs[tzid]; def != nil {
			if offset, ok := def.offsetAt(when); ok {
				// A whole-hour offset maps to a portable Etc/GMT zone so
				// recurrence can reconstruct it after the instant is persisted;
				// otherwise the name is empty and DST is not reproduced.
				return time.FixedZone(tzid, offset), etcZone(offset)
			}
		}
	}
	return time.UTC, ""
}

// etcZone returns the Etc/GMT zone name for a whole-hour offset (note Etc/GMT
// signs are inverted: Etc/GMT-10 is UTC+10), or "" when the offset cannot be
// named portably.
func etcZone(offset int) string {
	switch {
	case offset == 0:
		return "Etc/UTC"
	case offset%3600 != 0:
		return ""
	}
	hours := offset / 3600
	if hours < -14 || hours > 14 {
		return ""
	}
	if hours > 0 {
		return "Etc/GMT-" + strconv.Itoa(hours)
	}
	return "Etc/GMT+" + strconv.Itoa(-hours)
}

func parseTZDef(comp *ical.Component) *tzDef {
	def := &tzDef{}
	for _, sub := range comp.Children {
		switch sub.Name {
		case ical.CompTimezoneStandard, ical.CompTimezoneDaylight:
		default:
			continue
		}
		offProp := sub.Props.Get(ical.PropTimezoneOffsetTo)
		if offProp == nil {
			continue
		}
		offset, ok := parseUTCOffset(offProp.Value)
		if !ok {
			continue
		}
		def.subs = append(def.subs, tzSub{offset: offset, set: tzOnsetSet(sub)})
	}
	if len(def.subs) == 0 {
		return nil
	}
	return def
}

// tzOnsetSet collects a STANDARD/DAYLIGHT block's onset times (DTSTART plus any
// RRULE/RDATE), all read as floating wall-clock so they order consistently
// against the event's own wall-clock.
func tzOnsetSet(sub *ical.Component) *rrule.Set {
	startProp := sub.Props.Get(ical.PropDateTimeStart)
	if startProp == nil {
		return nil
	}
	start, err := time.ParseInLocation("20060102T150405", startProp.Value, time.UTC)
	if err != nil {
		return nil
	}

	set := &rrule.Set{}
	set.DTStart(start)
	for _, line := range rawValues(sub, ical.PropRecurrenceRule) {
		opt, err := rrule.StrToROptionInLocation(line, time.UTC)
		if err != nil {
			continue
		}
		opt.Dtstart = start
		rule, err := rrule.NewRRule(*opt)
		if err != nil {
			continue
		}
		set.RRule(rule)
	}
	for _, value := range rawValues(sub, ical.PropRecurrenceDates) {
		for raw := range strings.SplitSeq(value, ",") {
			if t, err := time.ParseInLocation("20060102T150405", strings.TrimSpace(raw), time.UTC); err == nil {
				set.RDate(t)
			}
		}
	}
	return set
}

// offsetAt returns the offset in effect at wall-clock when: the block whose most
// recent onset on-or-before when is latest. Before any transition, the
// first-declared block applies.
func (d *tzDef) offsetAt(when time.Time) (int, bool) {
	if len(d.subs) == 1 {
		return d.subs[0].offset, true
	}

	best := time.Time{}
	offset := 0
	found := false
	for _, sub := range d.subs {
		if sub.set == nil {
			continue
		}
		onset := sub.set.Before(when, true)
		if onset.IsZero() {
			continue
		}
		if !found || onset.After(best) {
			best, offset, found = onset, sub.offset, true
		}
	}
	if !found {
		return d.subs[0].offset, true
	}
	return offset, true
}

// parseUTCOffset parses an iCalendar UTC-offset value (+HHMM, -HHMM, ±HHMMSS).
func parseUTCOffset(v string) (int, bool) {
	v = strings.TrimSpace(v)
	if len(v) < 5 {
		return 0, false
	}
	sign := 1
	switch v[0] {
	case '+':
	case '-':
		sign = -1
	default:
		return 0, false
	}
	hh, err := strconv.Atoi(v[1:3])
	if err != nil {
		return 0, false
	}
	mm, err := strconv.Atoi(v[3:5])
	if err != nil {
		return 0, false
	}
	ss := 0
	if len(v) >= 7 {
		if ss, err = strconv.Atoi(v[5:7]); err != nil {
			return 0, false
		}
	}
	return sign * (hh*3600 + mm*60 + ss), true
}
