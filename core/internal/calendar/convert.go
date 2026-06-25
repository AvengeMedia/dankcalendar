package calendar

func (a Attendee) ToMap() map[string]any {
	out := map[string]any{"email": a.Email}
	if a.DisplayName != "" {
		out["displayName"] = a.DisplayName
	}
	if a.Role != "" {
		out["role"] = a.Role
	}
	if a.Status != "" {
		out["status"] = a.Status
	}
	if a.Optional {
		out["optional"] = true
	}
	if a.Organizer {
		out["organizer"] = true
	}
	return out
}

func AttendeesToMaps(attendees []Attendee) []map[string]any {
	if len(attendees) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(attendees))
	for _, a := range attendees {
		out = append(out, a.ToMap())
	}
	return out
}

// AttendeeFromMap reverses Attendee.ToMap after a JSON column round trip.
func AttendeeFromMap(m map[string]any) Attendee {
	a := Attendee{}
	a.Email, _ = m["email"].(string)
	a.DisplayName, _ = m["displayName"].(string)
	a.Role, _ = m["role"].(string)
	a.Status, _ = m["status"].(string)
	a.Optional, _ = m["optional"].(bool)
	a.Organizer, _ = m["organizer"].(bool)
	return a
}

func AttendeesFromMaps(maps []map[string]any) []Attendee {
	if len(maps) == 0 {
		return nil
	}
	out := make([]Attendee, 0, len(maps))
	for _, m := range maps {
		out = append(out, AttendeeFromMap(m))
	}
	return out
}

// OrganizerFromMap rebuilds the organizer attendee from its JSON column.
func OrganizerFromMap(m map[string]any) *Attendee {
	if len(m) == 0 {
		return nil
	}
	a := AttendeeFromMap(m)
	a.Organizer = true
	return &a
}

func RemindersToMaps(reminders []Reminder) []map[string]any {
	if len(reminders) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(reminders))
	for _, r := range reminders {
		out = append(out, map[string]any{"method": r.Method, "minutes": r.Minutes})
	}
	return out
}

// RemindersFromMaps reverses RemindersToMaps after a JSON column round trip,
// where minutes may come back as float64.
func RemindersFromMaps(maps []map[string]any) []Reminder {
	if len(maps) == 0 {
		return nil
	}
	out := make([]Reminder, 0, len(maps))
	for _, m := range maps {
		method, _ := m["method"].(string)
		var minutes int
		switch v := m["minutes"].(type) {
		case int:
			minutes = v
		case int64:
			minutes = int(v)
		case float64:
			minutes = int(v)
		default:
			continue
		}
		out = append(out, Reminder{Method: method, Minutes: minutes})
	}
	return out
}

// RecurrenceFromMap reverses Recurrence.ToMap after a JSON column round trip,
// where each rule list may come back as []any.
func RecurrenceFromMap(m map[string]any) *Recurrence {
	if len(m) == 0 {
		return nil
	}
	rec := &Recurrence{
		RRule:  stringSlice(m["rrule"]),
		RDate:  stringSlice(m["rdate"]),
		ExDate: stringSlice(m["exdate"]),
	}
	if len(rec.RRule)+len(rec.RDate)+len(rec.ExDate) == 0 {
		return nil
	}
	return rec
}

func stringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (r *Recurrence) ToMap() map[string]any {
	if r == nil {
		return nil
	}
	out := map[string]any{}
	if len(r.RRule) > 0 {
		out["rrule"] = r.RRule
	}
	if len(r.RDate) > 0 {
		out["rdate"] = r.RDate
	}
	if len(r.ExDate) > 0 {
		out["exdate"] = r.ExDate
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
