package calendar

import "strings"

// Canonical RSVP response statuses for the current user's participation in an
// event. Providers speak different vocabularies (Google responseStatus,
// Microsoft Graph response, iCalendar PARTSTAT); these are the normalized
// values stored and exchanged internally.
const (
	ResponseNeedsAction = "needs-action"
	ResponseAccepted    = "accepted"
	ResponseDeclined    = "declined"
	ResponseTentative   = "tentative"
)

// NormalizeResponse folds a provider-specific (or CLI verb) participation
// status onto the canonical set. Unknown or empty values are treated as
// needs-action so an unanswered invite is never mistaken for a reply.
func NormalizeResponse(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "accepted", "accept":
		return ResponseAccepted
	case "declined", "decline", "rejected", "reject":
		return ResponseDeclined
	case "tentative", "tentativelyaccepted", "tentativelyaccept", "maybe":
		return ResponseTentative
	case "organizer":
		// The organizer's own status is implicitly accepted.
		return ResponseAccepted
	default:
		return ResponseNeedsAction
	}
}

// CanRespond reports whether a canonical response represents an answer the user
// can submit (as opposed to the unanswered default).
func CanRespond(response string) bool {
	switch response {
	case ResponseAccepted, ResponseDeclined, ResponseTentative:
		return true
	default:
		return false
	}
}

// SelfEmail returns the current user's email for this account, used to find the
// user among an event's attendees. Google and Microsoft store the email as the
// account ID; CalDAV keeps the login in settings. Other account kinds have no
// notion of a personal identity and return "".
func (a Account) SelfEmail() string {
	switch a.Kind {
	case AccountGoogle, AccountMicrosoft:
		return normalizeEmail(a.ID)
	case AccountCalDAV:
		if a.Settings != nil {
			if u, ok := a.Settings["username"].(string); ok {
				return normalizeEmail(u)
			}
		}
	}
	return ""
}

// SelfAttendeeIndex returns the index of the attendee matching selfEmail, or -1
// when the user is not in the list (or selfEmail is unknown).
func SelfAttendeeIndex(attendees []Attendee, selfEmail string) int {
	self := normalizeEmail(selfEmail)
	if self == "" {
		return -1
	}
	for i, a := range attendees {
		if normalizeEmail(a.Email) == self {
			return i
		}
	}
	return -1
}

// SelfResponse reports the current user's normalized RSVP status for an event
// and whether they are a non-organizer attendee who may respond. Organizers and
// non-attendees return canRespond=false.
func SelfResponse(ev *Event, selfEmail string) (status string, canRespond bool) {
	self := normalizeEmail(selfEmail)
	if self == "" {
		return "", false
	}
	if ev.Organizer != nil && normalizeEmail(ev.Organizer.Email) == self {
		return ResponseAccepted, false
	}
	idx := SelfAttendeeIndex(ev.Attendees, selfEmail)
	if idx < 0 {
		return "", false
	}
	a := ev.Attendees[idx]
	if a.Organizer {
		return ResponseAccepted, false
	}
	return NormalizeResponse(a.Status), true
}

func normalizeEmail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 7 && strings.EqualFold(s[:7], "mailto:") {
		s = s[7:]
	}
	return strings.ToLower(strings.TrimSpace(s))
}
