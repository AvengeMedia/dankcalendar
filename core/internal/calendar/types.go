package calendar

import (
	"slices"
	"time"
)

type AccountKind string

const (
	AccountLocal     AccountKind = "local"
	AccountGoogle    AccountKind = "google"
	AccountCalDAV    AccountKind = "caldav"
	AccountMicrosoft AccountKind = "microsoft"
	AccountICal      AccountKind = "ical"
	AccountEvolution AccountKind = "evolution"
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
	ID                  string    `json:"id"`
	AccountID           string    `json:"accountId"`
	RemoteID            string    `json:"remoteId"`
	Name                string    `json:"name"`
	Description         string    `json:"description,omitempty"`
	Color               string    `json:"color,omitempty"`
	TimeZone            string    `json:"timeZone,omitempty"`
	ReadOnly            bool      `json:"readOnly"`
	Hidden              bool      `json:"hidden"`
	SyncToken           string    `json:"syncToken,omitempty"`
	SupportedComponents []string  `json:"supportedComponents,omitempty"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// ComponentVEvent and ComponentVTodo are the iCalendar component types a
// calendar collection may hold. A calendar with no declared components is
// treated as an event calendar for back-compat.
const (
	ComponentVEvent = "VEVENT"
	ComponentVTodo  = "VTODO"
)

// HoldsTasks reports whether the calendar is a task list (VTODO collection).
func (c Calendar) HoldsTasks() bool {
	return slices.Contains(c.SupportedComponents, ComponentVTodo)
}

// HoldsEvents reports whether the calendar holds events. A calendar with no
// declared components defaults to events.
func (c Calendar) HoldsEvents() bool {
	if len(c.SupportedComponents) == 0 {
		return true
	}
	return slices.Contains(c.SupportedComponents, ComponentVEvent)
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
	MeetingURL    string      `json:"meetingUrl,omitempty"`
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

type TaskStatus string

const (
	TaskNeedsAction TaskStatus = "needs_action"
	TaskInProcess   TaskStatus = "in_process"
	TaskCompleted   TaskStatus = "completed"
	TaskCancelled   TaskStatus = "cancelled"
)

// Task is an iCalendar VTODO. Unlike an event it has no fixed span: DUE and an
// optional DTSTART bound it, COMPLETED records when it was finished, and
// ParentUID links a subtask to its parent (RELATED-TO).
type Task struct {
	ID              string      `json:"id"`
	CalendarID      string      `json:"calendarId"`
	UID             string      `json:"uid"`
	RemoteID        string      `json:"remoteId,omitempty"`
	Etag            string      `json:"etag,omitempty"`
	Summary         string      `json:"summary"`
	Description     string      `json:"description,omitempty"`
	Location        string      `json:"location,omitempty"`
	Status          TaskStatus  `json:"status,omitempty"`
	Priority        int         `json:"priority,omitempty"`
	PercentComplete int         `json:"percentComplete,omitempty"`
	Due             time.Time   `json:"due,omitzero"`
	Start           time.Time   `json:"start,omitzero"`
	Completed       time.Time   `json:"completed,omitzero"`
	AllDay          bool        `json:"allDay"`
	DueTimeZone     string      `json:"dueTimeZone,omitempty"`
	StartTimeZone   string      `json:"startTimeZone,omitempty"`
	ParentUID       string      `json:"parentUid,omitempty"`
	Recurrence      *Recurrence `json:"recurrence,omitempty"`
	Categories      []string    `json:"categories,omitempty"`
	Reminders       []Reminder  `json:"reminders,omitempty"`
	RawICS          string      `json:"-"`
	Created         time.Time   `json:"created,omitempty"`
	Updated         time.Time   `json:"updated,omitempty"`
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

type TaskChange struct {
	Type     ChangeType `json:"type"`
	Task     *Task      `json:"task,omitempty"`
	RemoteID string     `json:"remoteId,omitempty"`
}

// FullSnapshot marks the result as a complete listing of the calendar:
// after applying all pages, events absent from the snapshot are pruned.
//
// RetryAfter is a provider hint for when it wants to be synced next. Zero means
// the engine picks its default interval.
//
// Color carries a calendar color discovered while syncing, for providers whose
// listing has no color source (ical feeds, local files). Empty leaves the
// stored color untouched.
type SyncResult struct {
	Cursor       SyncCursor    `json:"cursor"`
	Changes      []EventChange `json:"changes"`
	TaskChanges  []TaskChange  `json:"taskChanges,omitempty"`
	More         bool          `json:"more"`
	FullSnapshot bool          `json:"fullSnapshot"`
	RetryAfter   time.Duration `json:"retryAfter,omitempty"`
	Color        string        `json:"color,omitempty"`
}

type ListEventsOptions struct {
	Start time.Time
	End   time.Time
	Limit int
	Query string
}

type ListTasksOptions struct {
	IncludeCompleted bool
	Limit            int
	Query            string
}
