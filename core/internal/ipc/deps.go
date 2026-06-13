package ipc

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankcalendar/core/internal/reminders"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type SyncTrigger interface {
	SyncAccount(ctx context.Context, acc *ent.Account) error
	SyncAll(ctx context.Context) error
}

type RemindersEngine interface {
	Upcoming(ctx context.Context, limit int) ([]reminders.Upcoming, error)
	SendTest() error
}

type Deps struct {
	Repo      *repo.Repo
	Registry  *calendar.Registry
	Secrets   calendar.SecretStore
	Broker    *oauth.CallbackBroker
	Flows     *oauth.FlowRegistry
	HTTPAddr  string
	Sync      SyncTrigger
	Reminders RemindersEngine
	Bus       *EventBus
	Version   string
}
