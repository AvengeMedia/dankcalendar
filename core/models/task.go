package models

import "time"

type Task struct {
	ID              string     `json:"id"`
	CalendarID      string     `json:"calendarId"`
	UID             string     `json:"uid"`
	Summary         string     `json:"summary"`
	Description     string     `json:"description,omitempty"`
	Location        string     `json:"location,omitempty"`
	Status          string     `json:"status,omitempty"`
	Priority        int        `json:"priority,omitempty"`
	PercentComplete int        `json:"percentComplete,omitempty"`
	Due             *time.Time `json:"due,omitempty"`
	Start           *time.Time `json:"start,omitempty"`
	Completed       *time.Time `json:"completed,omitempty"`
	AllDay          bool       `json:"allDay"`
	ParentUID       string     `json:"parentUid,omitempty"`
}

type TaskList struct {
	Tasks []Task `json:"tasks"`
	Total int    `json:"total"`
}
