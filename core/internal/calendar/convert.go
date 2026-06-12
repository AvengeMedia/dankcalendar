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
