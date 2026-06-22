package calendar

import "regexp"

var meetingURLPattern = regexp.MustCompile(`https?://[^\s<>"]*(?:` + meetingHosts + `)[^\s<>"]*`)

const meetingHosts = `meet\.google\.com|` +
	`zoom\.us|zoomgov\.com|` +
	`teams\.microsoft\.com|teams\.live\.com|` +
	`webex\.com|` +
	`whereby\.com|` +
	`meet\.jit\.si|` +
	`chime\.aws|` +
	`bluejeans\.com|` +
	`gotomeeting\.com|gotomeet\.me`

// MeetingURLInText returns the first known conferencing join link found across
// the given texts, or "" when none match. Used as a fallback when a provider
// does not expose the meeting URL in a dedicated field.
func MeetingURLInText(texts ...string) string {
	for _, t := range texts {
		if m := meetingURLPattern.FindString(t); m != "" {
			return m
		}
	}
	return ""
}
