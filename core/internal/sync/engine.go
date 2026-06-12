package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	entaccount "github.com/AvengeMedia/dankcalendar/core/ent/account"
	enteventpkg "github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type Notifier func(topic string, data any)

type Engine struct {
	repo     *repo.Repo
	registry *calendar.Registry
	secrets  calendar.SecretStore
	interval time.Duration
	notify   Notifier

	mu      sync.Mutex
	running bool
	stop    chan struct{}
}

func NewEngine(r *repo.Repo, registry *calendar.Registry, secrets calendar.SecretStore, interval time.Duration) *Engine {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Engine{
		repo:     r,
		registry: registry,
		secrets:  secrets,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (e *Engine) SetNotifier(n Notifier) { e.notify = n }

func (e *Engine) publish(topic string, data any) {
	if e.notify == nil {
		return
	}
	e.notify(topic, data)
}

func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	go e.loop(ctx)
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}
	close(e.stop)
	e.running = false
	e.stop = make(chan struct{})
}

func (e *Engine) loop(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	if err := e.SyncAll(ctx); err != nil {
		log.Warnf("sync error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stop:
			return
		case <-ticker.C:
			if err := e.SyncAll(ctx); err != nil {
				log.Warnf("sync error: %v", err)
			}
		}
	}
}

func (e *Engine) SyncAll(ctx context.Context) error {
	accounts, err := e.repo.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}

	for _, acc := range accounts {
		if err := e.SyncAccount(ctx, acc); err != nil {
			log.Warnf("account %s sync error: %v", acc.ID, err)
		}
	}
	return nil
}

func (e *Engine) SyncAccount(ctx context.Context, acc *ent.Account) error {
	provider, err := e.registry.Build(ctx, accountToDomain(acc), e.secrets)
	if err != nil {
		return err
	}
	defer provider.Close()

	remoteCals, err := provider.ListCalendars(ctx)
	if err != nil {
		return fmt.Errorf("list calendars: %w", err)
	}

	for _, rc := range remoteCals {
		stored, err := e.repo.UpsertCalendar(ctx, repo.UpsertCalendarInput{
			AccountID:   acc.ID,
			RemoteID:    rc.RemoteID,
			Name:        rc.Name,
			Description: rc.Description,
			Color:       rc.Color,
			TimeZone:    rc.TimeZone,
			ReadOnly:    rc.ReadOnly,
			Hidden:      rc.Hidden,
		})
		if err != nil {
			log.Warnf("upsert calendar %q: %v", rc.RemoteID, err)
			continue
		}

		rc.ID = stored.ID
		if err := e.syncCalendarEvents(ctx, provider, rc, stored.SyncToken); err != nil {
			log.Warnf("sync calendar %q: %v", rc.RemoteID, err)
		}
	}

	e.publish("sync", map[string]any{"type": "completed", "accountId": acc.ID})
	return nil
}

func (e *Engine) syncCalendarEvents(ctx context.Context, provider calendar.Provider, cal calendar.Calendar, token string) error {
	cursor := calendar.SyncCursor{CalendarID: cal.ID, Token: token}
	var (
		snapshot     bool
		snapshotUIDs []string
		changed      int
	)

	for {
		result, err := provider.Sync(ctx, cal, cursor)
		if err != nil {
			return err
		}

		if err := e.applyChanges(ctx, cal, result.Changes); err != nil {
			return err
		}
		changed += len(result.Changes)

		if result.FullSnapshot {
			snapshot = true
			for _, ch := range result.Changes {
				if ch.Type == calendar.ChangeUpsert && ch.Event != nil {
					snapshotUIDs = append(snapshotUIDs, ch.Event.UID)
				}
			}
		}

		if err := e.repo.SetCalendarSyncToken(ctx, cal.ID, result.Cursor.Token); err != nil {
			return fmt.Errorf("persist sync token: %w", err)
		}

		cursor = result.Cursor
		if result.More {
			continue
		}

		if snapshot {
			pruned, err := e.repo.DeleteEventsNotInUIDs(ctx, cal.ID, snapshotUIDs)
			if err != nil {
				return fmt.Errorf("prune events: %w", err)
			}
			changed += pruned
		}

		if changed > 0 {
			e.publish("events", map[string]any{"type": "changed", "calendarId": cal.ID})
		}
		return nil
	}
}

func (e *Engine) applyChanges(ctx context.Context, cal calendar.Calendar, changes []calendar.EventChange) error {
	for _, ch := range changes {
		switch ch.Type {
		case calendar.ChangeUpsert:
			if ch.Event == nil {
				continue
			}
			ev := ch.Event
			var organizer map[string]any
			if ev.Organizer != nil {
				organizer = ev.Organizer.ToMap()
			}
			_, err := e.repo.UpsertEvent(ctx, repo.UpsertEventInput{
				CalendarID:    cal.ID,
				UID:           ev.UID,
				RemoteID:      ev.RemoteID,
				Etag:          ev.Etag,
				Summary:       ev.Summary,
				Description:   ev.Description,
				Location:      ev.Location,
				URL:           ev.URL,
				Status:        toEntStatus(ev.Status),
				Start:         ev.Start,
				End:           ev.End,
				AllDay:        ev.AllDay,
				StartTZ:       ev.StartTimeZone,
				EndTZ:         ev.EndTimeZone,
				Recurrence:    ev.Recurrence.ToMap(),
				RecurringID:   ev.RecurringID,
				OriginalStart: ev.OriginalStart,
				Organizer:     organizer,
				Attendees:     calendar.AttendeesToMaps(ev.Attendees),
				Reminders:     calendar.RemindersToMaps(ev.Reminders),
				Categories:    ev.Categories,
				Transparency:  ev.Transparency,
				Visibility:    ev.Visibility,
				RawICS:        ev.RawICS,
			})
			if err != nil {
				return fmt.Errorf("upsert event %q: %w", ev.UID, err)
			}
		case calendar.ChangeDelete:
			if err := e.repo.DeleteEventByUID(ctx, cal.ID, ch.RemoteID); err != nil {
				return fmt.Errorf("delete event %q: %w", ch.RemoteID, err)
			}
		}
	}
	return nil
}

func accountToDomain(a *ent.Account) calendar.Account {
	return calendar.Account{
		ID:          a.ID,
		Kind:        calendar.AccountKind(a.Kind),
		DisplayName: a.DisplayName,
		Settings:    a.Settings,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

func toEntStatus(s calendar.EventStatus) enteventpkg.Status {
	switch s {
	case calendar.EventTentative:
		return enteventpkg.StatusTentative
	case calendar.EventCancelled:
		return enteventpkg.StatusCancelled
	}
	return enteventpkg.StatusConfirmed
}

var _ = entaccount.IDEQ
