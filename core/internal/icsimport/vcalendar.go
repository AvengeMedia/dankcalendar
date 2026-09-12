package icsimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
	"golang.org/x/net/html/charset"
)

// Unfold quoted-printable soft breaks before go-ical unfolds content lines.
func calendarData(data []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(strings.TrimPrefix(string(data), "\ufeff"), "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		head, _, ok := strings.Cut(line, ":")
		if ok && strings.Contains(strings.ToUpper(head), "ENCODING=QUOTED-PRINTABLE") {
			for strings.HasSuffix(line, "=") && i+1 < len(lines) {
				i++
				line = strings.TrimSuffix(line, "=") + strings.TrimLeft(lines[i], " \t")
			}
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\r\n"))
}

func normalizeVCalendar(cal *ical.Calendar) error {
	p := cal.Props.Get(ical.PropVersion)
	if p == nil || p.Value != "1.0" {
		return nil
	}
	if p := cal.Props.Get("DAYLIGHT"); p != nil && !strings.HasPrefix(strings.ToUpper(p.Value), "FALSE") {
		return fmt.Errorf("vCalendar DAYLIGHT rules are unsupported; export this calendar as iCalendar 2.0")
	}
	var zone *time.Location
	if p := cal.Props.Get("TZ"); p != nil {
		raw := strings.ReplaceAll(p.Value, ":", "")
		parsed, err := time.Parse("-0700", raw)
		if err != nil {
			return fmt.Errorf("invalid vCalendar TZ: %w", err)
		}
		_, offset := parsed.Zone()
		zone = time.FixedZone("vCalendar", offset)
	}
	for _, ev := range cal.Events() {
		for name, props := range ev.Props {
			for i := range props {
				p := &props[i]
				if strings.EqualFold(p.Params.Get("ENCODING"), "QUOTED-PRINTABLE") {
					decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(p.Value)))
					if err != nil {
						return fmt.Errorf("decode vCalendar %s: %w", name, err)
					}
					if encoding := p.Params.Get("CHARSET"); encoding != "" {
						reader, err := charset.NewReaderLabel(encoding, bytes.NewReader(decoded))
						if err != nil {
							return err
						}
						decoded, err = io.ReadAll(reader)
						if err != nil {
							return err
						}
					}
					p.SetText(string(decoded))
					delete(p.Params, "ENCODING")
					delete(p.Params, "CHARSET")
				}
			}
			ev.Props[name] = props
		}
		if ev.Props.Get(ical.PropUID) == nil {
			data, err := json.Marshal(ev.Component)
			if err != nil {
				return err
			}
			ev.Props.SetText(ical.PropUID, fmt.Sprintf("%x@vcalendar.dankcalendar", sha256.Sum256(data)))
		}
		for _, name := range []string{ical.PropDateTimeStart, ical.PropDateTimeEnd, ical.PropRecurrenceID, ical.PropRecurrenceDates, ical.PropExceptionDates} {
			for i := range ev.Props[name] {
				p := &ev.Props[name][i]
				values := strings.FieldsFunc(p.Value, func(r rune) bool { return r == ';' || r == ',' })
				for j, raw := range values {
					if zone == nil || len(raw) == 8 || strings.HasSuffix(raw, "Z") {
						continue
					}
					t, err := time.ParseInLocation("20060102T150405", raw, zone)
					if err != nil {
						return fmt.Errorf("invalid vCalendar %s: %w", name, err)
					}
					values[j] = t.UTC().Format("20060102T150405Z")
				}
				p.Value = strings.Join(values, ",")
			}
		}
		for i := range ev.Props[ical.PropRecurrenceRule] {
			p := &ev.Props[ical.PropRecurrenceRule][i]
			rule, err := legacyRule(p.Value)
			if err != nil {
				return err
			}
			p.Value = rule
		}
	}
	cal.Props.SetText(ical.PropVersion, "2.0")
	return nil
}
