package models

import "time"

type Event struct {
	ID            string     `json:"id"`
	CalendarID    string     `json:"calendarId"`
	UID           string     `json:"uid"`
	Summary       string     `json:"summary"`
	Description   string     `json:"description,omitempty"`
	Location      string     `json:"location,omitempty"`
	URL           string     `json:"url,omitempty"`
	Status        string     `json:"status,omitempty"`
	Start         time.Time  `json:"start"`
	End           time.Time  `json:"end"`
	AllDay        bool       `json:"allDay"`
	StartTimeZone string     `json:"startTimeZone,omitempty"`
	EndTimeZone   string     `json:"endTimeZone,omitempty"`
	RecurringID   string     `json:"recurringId,omitempty"`
	Recurrence    []string   `json:"recurrence,omitempty"`
	Attendees     []Attendee `json:"attendees,omitempty"`
	Categories    []string   `json:"categories,omitempty"`
}

type Attendee struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Status      string `json:"status,omitempty"`
}

type EventCreate struct {
	CalendarID  string    `json:"calendarId"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	AllDay      bool      `json:"allDay,omitempty"`
	Recurrence  []string  `json:"recurrence,omitempty"`
}

type EventUpdate struct {
	Summary     *string    `json:"summary,omitempty"`
	Description *string    `json:"description,omitempty"`
	Location    *string    `json:"location,omitempty"`
	Start       *time.Time `json:"start,omitempty"`
	End         *time.Time `json:"end,omitempty"`
	AllDay      *bool      `json:"allDay,omitempty"`
}

type EventList struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
}
