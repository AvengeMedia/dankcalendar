package server

import (
	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type EmptyInput struct{}

type DeletedResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type Server struct {
	Cfg      *config.Config
	Repo     *repo.Repo
	Registry *calendar.Registry
	Secrets  calendar.SecretStore
}
