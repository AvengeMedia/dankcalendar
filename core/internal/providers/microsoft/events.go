package microsoft

import (
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

type graphEvent struct {
	ID                   string               `json:"id,omitempty"`
	Type                 string               `json:"type,omitempty"`
	ChangeKey            string               `json:"changeKey,omitempty"`
	Subject              string               `json:"subject"`
	BodyPreview          string               `json:"bodyPreview,omitempty"`
	Body                 *graphBody           `json:"body,omitempty"`
	Location             *graphLocation       `json:"location,omitempty"`
	WebLink              string               `json:"webLink,omitempty"`
	IsCancelled          bool                 `json:"isCancelled,omitempty"`
	IsAllDay             bool                 `json:"isAllDay"`
	Start                *graphDateTime       `json:"start,omitempty"`
	End                  *graphDateTime       `json:"end,omitempty"`
	SeriesMasterID       string               `json:"seriesMasterId,omitempty"`
	Organizer            *graphRecipient      `json:"organizer,omitempty"`
	Attendees            []graphAttendee      `json:"attendees,omitempty"`
	ResponseStatus       *graphResponseStatus `json:"responseStatus,omitempty"`
	ShowAs               string               `json:"showAs,omitempty"`
	Sensitivity          string               `json:"sensitivity,omitempty"`
	CreatedDateTime      string               `json:"createdDateTime,omitempty"`
	LastModifiedDateTime string               `json:"lastModifiedDateTime,omitempty"`
	Removed              *graphRemoved        `json:"@removed,omitempty"`
}

func graphToEvent(g graphEvent) cal.Event {
	ev := cal.Event{
		UID:         g.ID,
		RemoteID:    g.ID,
		Etag:        g.ChangeKey,
		Summary:     g.Subject,
		Description: graphDescription(g),
		URL:         g.WebLink,
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
	return ev
}

// graphDescription prefers the full body over bodyPreview, which Graph
// truncates to ~255 chars. HTML bodies are converted to markdown so list
// and link structure survives; the UI renders descriptions as markdown.
func graphDescription(g graphEvent) string {
	if g.Body == nil {
		return g.BodyPreview
	}
	content := strings.TrimSpace(g.Body.Content)
	if content == "" {
		return g.BodyPreview
	}
	if !strings.EqualFold(g.Body.ContentType, "html") {
		return content
	}
	md, err := htmltomarkdown.ConvertString(content)
	if err != nil {
		return g.BodyPreview
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
		g.Attendees = append(g.Attendees, graphAttendee{
			EmailAddress: graphEmailAddress{Address: a.Email, Name: a.DisplayName},
			Type:         attendeeType,
		})
	}
	return g
}

func midnightUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
