package calendar

import (
	"context"
	"errors"
)

// ErrReauthRequired marks a sync failure as a dead credential: the stored
// token was rejected and the account must be re-authorized. Providers wrap it
// around the underlying error so the sync engine can flag the account.
var ErrReauthRequired = errors.New("reauthentication required")

type Provider interface {
	Kind() AccountKind
	Account() Account

	ListCalendars(ctx context.Context) ([]Calendar, error)
	Sync(ctx context.Context, cal Calendar, cursor SyncCursor) (*SyncResult, error)
	ListEvents(ctx context.Context, cal Calendar, opts ListEventsOptions) ([]Event, error)

	CreateEvent(ctx context.Context, cal Calendar, ev *Event) (*Event, error)
	UpdateEvent(ctx context.Context, cal Calendar, ev *Event) (*Event, error)
	DeleteEvent(ctx context.Context, cal Calendar, ev Event) error

	Close() error
}

type ProviderFactory interface {
	Kind() AccountKind
	Build(ctx context.Context, account Account, secrets SecretStore) (Provider, error)
}

type SecretStore interface {
	Get(ctx context.Context, accountID, key string) ([]byte, error)
	Set(ctx context.Context, accountID, key string, value []byte) error
	Delete(ctx context.Context, accountID, key string) error
}
