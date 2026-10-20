package server

import (
	"github.com/duxlabs-dev/dux-calendar/core/config"
	"github.com/duxlabs-dev/dux-calendar/core/internal/calendar"
	"github.com/duxlabs-dev/dux-calendar/core/repo"
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
