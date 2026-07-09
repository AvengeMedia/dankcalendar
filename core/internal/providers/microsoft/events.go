package microsoft

import (
	"strconv"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

const graphTimeLayout = "2006-01-02T15:04:05"

type graphDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type graphEmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

type graphRecipient struct {
	EmailAddress graphEmailAddress `json:"emailAddress"`
}

type graphResponseStatus struct {
	Response string `json:"response"`
}

type graphAttendee struct {
	EmailAddress graphEmailAddress    `json:"emailAddress"`
	Type         string               `json:"type,omitempty"`
	Status       *graphResponseStatus `json:"status,omitempty"`
}

type graphLocation struct {
	DisplayName string `json:"displayName"`
}

type graphBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type graphRemoved struct {
	Reason string `json:"reason"`
}

type graphOnlineMeeting struct {
	JoinURL string `json:"joinUrl,omitempty"`
}

type graphEvent struct {
	ID                         string                    `json:"id,omitempty"`
	Type                       string                    `json:"type,omitempty"`
	ChangeKey                  string                    `json:"changeKey,omitempty"`
	Subject                    string                    `json:"subject"`
	BodyPreview                string                    `json:"bodyPreview,omitempty"`
	Body                       *graphBody                `json:"body,omitempty"`
	Location                   *graphLocation            `json:"location,omitempty"`
	WebLink                    string                    `json:"webLink,omitempty"`
	OnlineMeetingURL           string                    `json:"onlineMeetingUrl,omitempty"`
	OnlineMeeting              *graphOnlineMeeting       `json:"onlineMeeting,omitempty"`
	IsCancelled                bool                      `json:"isCancelled,omitempty"`
	IsAllDay                   bool                      `json:"isAllDay"`
	Start                      *graphDateTime            `json:"start,omitempty"`
	End                        *graphDateTime            `json:"end,omitempty"`
	SeriesMasterID             string                    `json:"seriesMasterId,omitempty"`
	Recurrence                 *graphPatternedRecurrence `json:"recurrence,omitempty"`
	IsReminderOn               *bool                     `json:"isReminderOn,omitempty"`
	ReminderMinutesBeforeStart *int                      `json:"reminderMinutesBeforeStart,omitempty"`
	Organizer                  *graphRecipient           `json:"organizer,omitempty"`
	Attendees                  []graphAttendee           `json:"attendees,omitempty"`
	ResponseStatus             *graphResponseStatus      `json:"responseStatus,omitempty"`
	ShowAs                     string                    `json:"showAs,omitempty"`
	Sensitivity                string                    `json:"sensitivity,omitempty"`
	CreatedDateTime            string                    `json:"createdDateTime,omitempty"`
	LastModifiedDateTime       string                    `json:"lastModifiedDateTime,omitempty"`
	Removed                    *graphRemoved             `json:"@removed,omitempty"`
}

func graphToEvent(g graphEvent) cal.Event {
	ev := cal.Event{
		UID:         g.ID,
		RemoteID:    g.ID,
		Etag:        g.ChangeKey,
		Summary:     g.Subject,
		Description: graphDescription(g),
		URL:         g.WebLink,
		MeetingURL:  graphMeetingURL(g),
		Status:      graphStatus(g),
		AllDay:      g.IsAllDay,
		RecurringID: g.SeriesMasterID,
	}

	if g.Location != nil {
		ev.Location = g.Location.DisplayName
	}
	if g.Start != nil {
		ev.Start = parseGraphTime(g.Start.DateTime)
	}
	if g.End != nil {
		ev.End = parseGraphTime(g.End.DateTime)
	}

	if g.Organizer != nil && g.Organizer.EmailAddress.Address != "" {
		ev.Organizer = &cal.Attendee{
			Email:       g.Organizer.EmailAddress.Address,
			DisplayName: g.Organizer.EmailAddress.Name,
			Organizer:   true,
		}
	}

	for _, a := range g.Attendees {
		att := cal.Attendee{
			Email:       a.EmailAddress.Address,
			DisplayName: a.EmailAddress.Name,
			Optional:    a.Type == "optional",
		}
		if a.Status != nil {
			att.Status = a.Status.Response
		}
		ev.Attendees = append(ev.Attendees, att)
	}

	switch g.ShowAs {
	case "free":
		ev.Transparency = "transparent"
	default:
		ev.Transparency = "opaque"
	}

	if g.Sensitivity != "" && g.Sensitivity != "normal" {
		ev.Visibility = g.Sensitivity
	}

	if t, err := time.Parse(time.RFC3339, g.CreatedDateTime); err == nil {
		ev.Created = t
	}
	if t, err := time.Parse(time.RFC3339, g.LastModifiedDateTime); err == nil {
		ev.Updated = t
	}

	if g.IsReminderOn != nil && *g.IsReminderOn && g.ReminderMinutesBeforeStart != nil {
		ev.Reminders = []cal.Reminder{{Method: "popup", Minutes: *g.ReminderMinutesBeforeStart}}
	}
	return ev
}

// graphMeetingURL prefers the structured onlineMeeting.joinUrl (set only for
// Teams), then the flat onlineMeetingUrl, then a known link in the body or
// location — where Zoom and other add-ins put their join URL.
func graphMeetingURL(g graphEvent) string {
	if g.OnlineMeeting != nil && g.OnlineMeeting.JoinURL != "" {
		return g.OnlineMeeting.JoinURL
	}
	if g.OnlineMeetingURL != "" {
		return g.OnlineMeetingURL
	}
	var location string
	if g.Location != nil {
		location = g.Location.DisplayName
	}
	return cal.MeetingURLInText(graphDescription(g), location)
}

// graphDescription prefers the full body over bodyPreview, which Graph
// truncates to ~255 chars. HTML bodies are converted to markdown so list
// and link structure survives; the UI renders descriptions as markdown.
func graphDescription(g graphEvent) string {
	if text := graphBodyText(g.Body); text != "" {
		return text
	}
	return g.BodyPreview
}

// graphBodyText extracts a plain/markdown description, converting HTML bodies so
// Outlook markup doesn't leak through. Empty on nil/blank/failed conversion.
func graphBodyText(b *graphBody) string {
	if b == nil {
		return ""
	}
	content := strings.TrimSpace(b.Content)
	if content == "" || !strings.EqualFold(b.ContentType, "html") {
		return content
	}
	md, err := htmltomarkdown.ConvertString(content)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(md)
}

func graphStatus(g graphEvent) cal.EventStatus {
	switch {
	case g.IsCancelled:
		return cal.EventCancelled
	case g.ResponseStatus != nil && g.ResponseStatus.Response == "tentativelyAccepted":
		return cal.EventTentative
	default:
		return cal.EventConfirmed
	}
}

func parseGraphTime(s string) time.Time {
	base, _, _ := strings.Cut(s, ".")
	t, err := time.ParseInLocation(graphTimeLayout, base, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}

func eventToGraph(ev *cal.Event) graphEvent {
	g := graphEvent{
		Subject:  ev.Summary,
		Body:     &graphBody{ContentType: "text", Content: ev.Description},
		IsAllDay: ev.AllDay,
	}
	if ev.Location != "" {
		g.Location = &graphLocation{DisplayName: ev.Location}
	}

	start, end := ev.Start.UTC(), ev.End.UTC()
	if ev.AllDay {
		start = midnightUTC(start)
		end = midnightUTC(end)
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
	}
	g.Start = &graphDateTime{DateTime: start.Format(graphTimeLayout), TimeZone: "UTC"}
	g.End = &graphDateTime{DateTime: end.Format(graphTimeLayout), TimeZone: "UTC"}

	for _, a := range ev.Attendees {
		attendeeType := "required"
		if a.Optional {
			attendeeType = "optional"
		}
		ga := graphAttendee{
			EmailAddress: graphEmailAddress{Address: a.Email, Name: a.DisplayName},
			Type:         attendeeType,
		}
		if resp := graphResponse(a.Status); resp != "" {
			ga.Status = &graphResponseStatus{Response: resp}
		}
		g.Attendees = append(g.Attendees, ga)
	}

	g.Recurrence = recurrenceToGraph(ev)
	if minutes, ok := earliestPopupReminder(ev.Reminders); ok {
		on := true
		g.IsReminderOn, g.ReminderMinutesBeforeStart = &on, &minutes
	} else {
		off := false
		g.IsReminderOn = &off
	}
	return g
}

// recurrenceToGraph converts the event's RRULE into a Graph patterned
// recurrence. Rules outside Graph's pattern vocabulary (sub-daily FREQ,
// ordinal BYDAY, extra BY* parts, multiple rules) return nil so the event
// is written as a single occurrence rather than recurring wrongly.
func recurrenceToGraph(ev *cal.Event) *graphPatternedRecurrence {
	if ev.Recurrence == nil || len(ev.Recurrence.RRule) != 1 {
		return nil
	}
	rule := parseRRule(ev.Recurrence.RRule[0])
	for key := range rule {
		switch key {
		case "FREQ", "INTERVAL", "BYDAY", "COUNT", "UNTIL", "WKST":
		default:
			return nil
		}
	}
	if rule["BYDAY"] != "" {
		if rule["FREQ"] != "WEEKLY" {
			return nil
		}
		for _, code := range strings.Split(rule["BYDAY"], ",") {
			switch strings.TrimSpace(code) {
			case "SU", "MO", "TU", "WE", "TH", "FR", "SA":
			default:
				return nil
			}
		}
	}

	start := ev.Start.UTC()
	pattern := graphRecurrencePattern{Interval: 1}
	if rule["INTERVAL"] != "" {
		n, err := strconv.Atoi(rule["INTERVAL"])
		if err != nil || n < 1 {
			return nil
		}
		pattern.Interval = n
	}
	switch rule["FREQ"] {
	case "DAILY":
		pattern.Type = "daily"
	case "WEEKLY":
		pattern.Type = "weekly"
		pattern.DaysOfWeek = graphWeekdaysFromRRule(rule["BYDAY"], start.Weekday())
	case "MONTHLY":
		pattern.Type = "absoluteMonthly"
		pattern.DayOfMonth = start.Day()
	case "YEARLY":
		pattern.Type = "absoluteYearly"
		pattern.Month = int(start.Month())
		pattern.DayOfMonth = start.Day()
	default:
		return nil
	}

	rng := graphRecurrenceRange{Type: "noEnd", StartDate: start.Format("2006-01-02"), RecurrenceTimeZone: "UTC"}
	switch {
	case rule["COUNT"] != "":
		n, err := strconv.Atoi(rule["COUNT"])
		if err != nil || n < 1 {
			return nil
		}
		rng.Type = "numbered"
		rng.NumberOfOccurrences = n
	case rule["UNTIL"] != "":
		t, ok := parseRRuleUntil(rule["UNTIL"])
		if !ok {
			return nil
		}
		rng.Type = "endDate"
		rng.EndDate = t.UTC().Format("2006-01-02")
	}
	return &graphPatternedRecurrence{Pattern: pattern, Range: rng}
}

func parseRRuleUntil(value string) (time.Time, bool) {
	for _, layout := range []string{"20060102", "20060102T150405Z"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// earliestPopupReminder picks the popup reminder that fires first — Graph
// supports a single reminder offset. The rest still fire locally.
func earliestPopupReminder(reminders []cal.Reminder) (int, bool) {
	minutes, found := 0, false
	for _, r := range reminders {
		switch r.Method {
		case "", "popup", "display":
		default:
			continue
		}
		if r.Minutes < 0 {
			continue
		}
		if !found || r.Minutes > minutes {
			minutes, found = r.Minutes, true
		}
	}
	return minutes, found
}

// graphResponse maps a stored participation status onto the Graph response
// vocabulary, returning "" for an unanswered status so it is left unset.
func graphResponse(status string) string {
	switch cal.NormalizeResponse(status) {
	case cal.ResponseAccepted:
		return "accepted"
	case cal.ResponseDeclined:
		return "declined"
	case cal.ResponseTentative:
		return "tentativelyAccepted"
	default:
		return ""
	}
}

func midnightUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
