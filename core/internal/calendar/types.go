package calendar

import "time"

type AccountKind string

const (
	AccountLocal     AccountKind = "local"
	AccountGoogle    AccountKind = "google"
	AccountCalDAV    AccountKind = "caldav"
	AccountMicrosoft AccountKind = "microsoft"
)

type Account struct {
	ID          string         `json:"id"`
	Kind        AccountKind    `json:"kind"`
	DisplayName string         `json:"displayName"`
	Settings    map[string]any `json:"settings,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type Calendar struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"accountId"`
	RemoteID    string    `json:"remoteId"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Color       string    `json:"color,omitempty"`
	TimeZone    string    `json:"timeZone,omitempty"`
	ReadOnly    bool      `json:"readOnly"`
	Hidden      bool      `json:"hidden"`
	SyncToken   string    `json:"syncToken,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type EventStatus string

const (
	EventConfirmed EventStatus = "confirmed"
	EventTentative EventStatus = "tentative"
	EventCancelled EventStatus = "cancelled"
)

type Attendee struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Role        string `json:"role,omitempty"`
	Status      string `json:"status,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Organizer   bool   `json:"organizer,omitempty"`
}

type Reminder struct {
	Method  string `json:"method"`
	Minutes int    `json:"minutes"`
}

type Recurrence struct {
	RRule  []string `json:"rrule,omitempty"`
	RDate  []string `json:"rdate,omitempty"`
	ExDate []string `json:"exdate,omitempty"`
}

type Event struct {
	ID            string      `json:"id"`
	CalendarID    string      `json:"calendarId"`
	UID           string      `json:"uid"`
	RemoteID      string      `json:"remoteId,omitempty"`
	Etag          string      `json:"etag,omitempty"`
	Summary       string      `json:"summary"`
	Description   string      `json:"description,omitempty"`
	Location      string      `json:"location,omitempty"`
	URL           string      `json:"url,omitempty"`
	Status        EventStatus `json:"status,omitempty"`
	Start         time.Time   `json:"start"`
	End           time.Time   `json:"end"`
	AllDay        bool        `json:"allDay"`
	StartTimeZone string      `json:"startTimeZone,omitempty"`
	EndTimeZone   string      `json:"endTimeZone,omitempty"`
	Recurrence    *Recurrence `json:"recurrence,omitempty"`
	RecurringID   string      `json:"recurringId,omitempty"`
	OriginalStart time.Time   `json:"originalStart,omitzero"`
	Organizer     *Attendee   `json:"organizer,omitempty"`
	Attendees     []Attendee  `json:"attendees,omitempty"`
	Reminders     []Reminder  `json:"reminders,omitempty"`
	Categories    []string    `json:"categories,omitempty"`
	Transparency  string      `json:"transparency,omitempty"`
	Visibility    string      `json:"visibility,omitempty"`
	RawICS        string      `json:"-"`
	Created       time.Time   `json:"created,omitempty"`
	Updated       time.Time   `json:"updated,omitempty"`
}

type EventOccurrence struct {
	Event
	OccurrenceStart time.Time `json:"occurrenceStart"`
	OccurrenceEnd   time.Time `json:"occurrenceEnd"`
}

type SyncCursor struct {
	CalendarID string `json:"calendarId"`
	Token      string `json:"token,omitempty"`
}

type ChangeType string

const (
	ChangeUpsert ChangeType = "upsert"
	ChangeDelete ChangeType = "delete"
)

type EventChange struct {
	Type     ChangeType `json:"type"`
	Event    *Event     `json:"event,omitempty"`
	RemoteID string     `json:"remoteId,omitempty"`
}

// FullSnapshot marks the result as a complete listing of the calendar:
// after applying all pages, events absent from the snapshot are pruned.
type SyncResult struct {
	Cursor       SyncCursor    `json:"cursor"`
	Changes      []EventChange `json:"changes"`
	More         bool          `json:"more"`
	FullSnapshot bool          `json:"fullSnapshot"`
}

type ListEventsOptions struct {
	Start time.Time
	End   time.Time
	Limit int
	Query string
}
