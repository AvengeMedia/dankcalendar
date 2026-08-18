// Package rsvp submits the current user's accept/decline/tentative reply to a
// synced event: it pushes the response to the originating provider and persists
// the new participation status. Shared by the IPC handler and the invitation
// notification engine so both reply identically.
package rsvp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/eventconv"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type Stores struct {
	Repo     *repo.Repo
	Registry *calendar.Registry
	Secrets  calendar.SecretStore
}

type Result struct {
	EventID    string
	CalendarID string
	Response   string
	Summary    string
}

// ErrNotAttendee marks the user as absent from the event's attendee list, so
// there is no response to submit.
var ErrNotAttendee = errors.New("you are not an attendee of this event")

// Apply submits response (any NormalizeResponse-able value) for the event. A
// non-zero occurrenceStart replies for that single occurrence of a recurring
// series instead of the whole series; it is ignored for non-recurring events.
func Apply(ctx context.Context, s Stores, eventID, response string, occurrenceStart time.Time) (*Result, error) {
	canonical := calendar.NormalizeResponse(response)
	if !calendar.CanRespond(canonical) {
		return nil, fmt.Errorf("response must be accept, decline, or tentative")
	}

	entEv, err := s.Repo.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if entEv.Edges.Calendar == nil {
		return nil, errors.New("event has no calendar")
	}

	entCal, err := s.Repo.GetCalendar(ctx, entEv.Edges.Calendar.ID)
	if err != nil {
		return nil, err
	}
	if entCal.ReadOnly {
		return nil, fmt.Errorf("calendar %q is read-only", entCal.Name)
	}
	entAcc := entCal.Edges.Account
	if entAcc == nil {
		return nil, errors.New("calendar has no account")
	}

	domAcc := calendar.Account{
		ID:          entAcc.ID,
		Kind:        calendar.AccountKind(entAcc.Kind),
		DisplayName: entAcc.DisplayName,
		Settings:    entAcc.Settings,
	}

	ev := eventconv.FromEnt(entEv)
	if !occurrenceStart.IsZero() && ev.Recurrence != nil {
		duration := ev.End.Sub(ev.Start)
		ev.RecurringID = ev.UID
		ev.OriginalStart = occurrenceStart
		ev.Start = occurrenceStart
		ev.End = occurrenceStart.Add(duration)
		ev.Recurrence = nil
	}
	idx := calendar.SelfAttendeeIndex(ev.Attendees, domAcc.SelfEmail())
	if idx < 0 {
		return nil, ErrNotAttendee
	}
	ev.Attendees[idx].Status = canonical

	provider, err := s.Registry.Build(ctx, domAcc, s.Secrets)
	if err != nil {
		return nil, err
	}
	defer provider.Close()

	domCal := calendar.Calendar{
		ID:        entCal.ID,
		AccountID: entAcc.ID,
		RemoteID:  entCal.RemoteID,
		Name:      entCal.Name,
		TimeZone:  entCal.TimeZone,
		ReadOnly:  entCal.ReadOnly,
	}

	updated, err := submit(ctx, provider, domCal, &ev, canonical)
	if err != nil {
		return nil, err
	}

	// Providers that do not echo attendees back (Microsoft) leave the reply for
	// us to stamp; the next sync reconciles the authoritative state.
	switch i := calendar.SelfAttendeeIndex(updated.Attendees, domAcc.SelfEmail()); {
	case i >= 0:
		updated.Attendees[i].Status = canonical
	case len(updated.Attendees) == 0:
		updated.Attendees = ev.Attendees
	}

	stored, err := s.Repo.UpsertEvent(ctx, eventconv.UpsertInput(domCal.ID, updated))
	if err != nil {
		return nil, err
	}

	return &Result{
		EventID:    stored.ID,
		CalendarID: domCal.ID,
		Response:   canonical,
		Summary:    stored.Summary,
	}, nil
}

// submit prefers a provider's native RSVP path, falling back to a full update
// that carries the changed PARTSTAT for iCalendar-based providers.
func submit(ctx context.Context, provider calendar.Provider, cal calendar.Calendar, ev *calendar.Event, response string) (*calendar.Event, error) {
	if r, ok := provider.(calendar.Responder); ok {
		return r.RespondToEvent(ctx, cal, ev, response)
	}
	return provider.UpdateEvent(ctx, cal, ev)
}
