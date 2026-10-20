package ipc

import (
	"context"

	"github.com/duxlabs-dev/dux-calendar/core/ent"
	"github.com/duxlabs-dev/dux-calendar/core/internal/calendar"
	"github.com/duxlabs-dev/dux-calendar/core/internal/colorscheme"
	"github.com/duxlabs-dev/dux-calendar/core/internal/oauth"
	"github.com/duxlabs-dev/dux-calendar/core/internal/reminders"
	"github.com/duxlabs-dev/dux-calendar/core/repo"
	"github.com/duxlabs-dev/dux-go/files"
)

type SyncTrigger interface {
	SyncAccount(ctx context.Context, acc *ent.Account) error
	SyncAll(ctx context.Context) error
}

type RemindersEngine interface {
	Upcoming(ctx context.Context, limit int) ([]reminders.Upcoming, error)
	SendTest() error
}

// URIOpener opens a URI with the user's default handler.
type URIOpener interface {
	OpenURI(uri string) error
}

type Deps struct {
	Repo        *repo.Repo
	Registry    *calendar.Registry
	Secrets     calendar.SecretStore
	Broker      *oauth.CallbackBroker
	Flows       *oauth.FlowRegistry
	HTTPAddr    string
	Sync        SyncTrigger
	Reminders   RemindersEngine
	Bus         *EventBus
	Pending     *PendingOpen
	Version     string
	ColorScheme *colorscheme.Watcher
	Opener      URIOpener
	Files       *files.Service
}
