package models

import "time"

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
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CalendarUpdate struct {
	Hidden *bool   `json:"hidden,omitempty"`
	Color  *string `json:"color,omitempty"`
}
