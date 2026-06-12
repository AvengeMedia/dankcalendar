package models

import "time"

type Account struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind" enum:"local,google,caldav"`
	DisplayName string         `json:"displayName"`
	Settings    map[string]any `json:"settings,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type AccountCreate struct {
	Kind        string         `json:"kind" enum:"local,google,caldav"`
	DisplayName string         `json:"displayName"`
	Settings    map[string]any `json:"settings,omitempty"`
}

type AccountUpdate struct {
	DisplayName *string        `json:"displayName,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
}
