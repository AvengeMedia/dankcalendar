// Package invitations watches synced calendar data for meeting invitations the
// user has not answered and prompts them with an accept/decline/tentative
// desktop notification. Each invitation notifies at most once, tracked via
// InvitationState rows; acting on a notification submits the RSVP through the
// shared rsvp package. Evaluation is event-driven: the engine wakes when
// calendar data changes and re-scans.
package invitations

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	entevent "github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/notify"
	"github.com/AvengeMedia/dankcalendar/core/internal/rsvp"
	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

const (
	// lookahead bounds how far ahead an unanswered invite is surfaced.
	lookahead = 30 * 24 * time.Hour
	// pruneAfter drops notify-once rows once the event is well in the past.
	pruneAfter = 60 * 24 * time.Hour
	// defaultMaxWake caps how long the engine parks between scans; a backstop
	// against a missed wake signal, not the normal path.
	defaultMaxWake = time.Hour
	// coalesce bounds how soon a wake may force a rescan. A sync pass can
	// rewrite thousands of Event rows individually, each firing Wake(); without
	// this, every single row would trigger its own full recurring-event
	// re-expansion. Wakes arriving faster than this just push the scan out.
	coalesce = 2 * time.Second
)

// Sender is the subset of the notify client the engine needs.
type Sender interface {
	Send(n notify.Notification) (uint32, error)
	Dismiss(id uint32)
}

type Publisher func(topic string, data any)

type Engine struct {
	repo     *repo.Repo
	stores   rsvp.Stores
	sender   Sender
	publish  Publisher
	settings func() settings.UISettings
	now      func() time.Time
	loc      *time.Location
	open     func()
	maxWake  time.Duration

	wake chan struct{}

	mu      sync.Mutex
	pending map[uint32]pendingInvite
	running bool
	stop    chan struct{}
}

type pendingInvite struct {
	eventID    string
	calendarID string
}

func NewEngine(r *repo.Repo, stores rsvp.Stores, sender Sender, maxWake time.Duration) *Engine {
	if maxWake <= 0 {
		maxWake = defaultMaxWake
	}
	return &Engine{
		repo:     r,
		stores:   stores,
		sender:   sender,
		settings: settings.Load,
		now:      time.Now,
		loc:      time.Local,
		open:     openApp,
		maxWake:  maxWake,
		wake:     make(chan struct{}, 1),
		pending:  make(map[uint32]pendingInvite),
		stop:     make(chan struct{}),
	}
}

func (e *Engine) SetPublisher(p Publisher) { e.publish = p }

func (e *Engine) Start(ctx context.Context) {
	if e.sender == nil {
		log.Warnf("invitations: no notification transport, engine not started")
		return
	}

	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	go e.loop(ctx)
}

// Wake triggers a rescan; concurrent calls coalesce.
func (e *Engine) Wake() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// WatchMutations rescans when event or calendar data changes. InvitationState
// is excluded: the engine writes it when firing, so hooking it would loop.
func (e *Engine) WatchMutations(client *ent.Client) {
	wake := func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			v, err := next.Mutate(ctx, m)
			if err == nil {
				e.Wake()
			}
			return v, err
		})
	}
	client.Event.Use(wake)
	client.Calendar.Use(wake)
	client.Account.Use(wake)
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
	if n, err := e.repo.PruneInvitationStates(ctx, e.now().Add(-pruneAfter)); err == nil && n > 0 {
		log.Debugf("invitations: pruned %d stale states", n)
	}

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stop:
			return
		case <-e.wake:
			resetTimer(timer, coalesce)
			continue
		case <-timer.C:
		}

		if err := e.Tick(ctx); err != nil {
			log.Warnf("invitations: %v", err)
		}
		resetTimer(timer, e.maxWake)
	}
}

func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// Tick scans upcoming events for unanswered invitations and fires a one-time
// notification for each.
func (e *Engine) Tick(ctx context.Context) error {
	if !e.settings().RemindersEnabled {
		return nil
	}

	now := e.now()
	selfByCalendar, err := e.selfEmails(ctx)
	if err != nil {
		return err
	}

	from, to := now, now.Add(lookahead)
	events, _, err := e.repo.ListEvents(ctx, repo.ListEventsParams{
		Filter: repo.EventFilter{From: &from, To: &to, IncludeRecurring: true},
	})
	if err != nil {
		return err
	}

	notified, err := e.notifiedSet(ctx, now)
	if err != nil {
		return err
	}

	for _, ev := range events {
		calID := eventCalendarID(ev)
		if calID == "" || ev.ID == "" {
			continue
		}
		if ev.Status == entevent.StatusCancelled {
			continue
		}
		if !ev.Start.After(now) {
			continue
		}
		selfEmail, ok := selfByCalendar[calID]
		if !ok || selfEmail == "" {
			continue
		}
		if _, seen := notified[stateKey{calID, ev.UID}]; seen {
			continue
		}

		status, canRespond := calendar.SelfResponse(domainSnapshot(ev), selfEmail)
		if !canRespond || status != calendar.ResponseNeedsAction {
			continue
		}
		e.fire(ctx, ev, calID, now)
	}
	return nil
}

// selfEmails maps each writable calendar to the account user's email, skipping
// read-only calendars and account kinds without a personal identity.
func (e *Engine) selfEmails(ctx context.Context) (map[string]string, error) {
	accounts, err := e.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	selfByAccount := make(map[string]string, len(accounts))
	for _, a := range accounts {
		domAcc := calendar.Account{ID: a.ID, Kind: calendar.AccountKind(a.Kind), Settings: a.Settings}
		selfByAccount[a.ID] = domAcc.SelfEmail()
	}

	cals, err := e.repo.ListCalendars(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(cals))
	for _, c := range cals {
		if c.ReadOnly || c.Edges.Account == nil {
			continue
		}
		if email := selfByAccount[c.Edges.Account.ID]; email != "" {
			out[c.ID] = email
		}
	}
	return out, nil
}

type stateKey struct {
	calendarID string
	uid        string
}

func (e *Engine) notifiedSet(ctx context.Context, now time.Time) (map[stateKey]struct{}, error) {
	states, err := e.repo.ListInvitationStates(ctx, now.Add(-pruneAfter), now.Add(lookahead))
	if err != nil {
		return nil, err
	}
	out := make(map[stateKey]struct{}, len(states))
	for _, st := range states {
		out[stateKey{st.CalendarID, st.UID}] = struct{}{}
	}
	return out, nil
}

func (e *Engine) fire(ctx context.Context, ev *ent.Event, calID string, now time.Time) {
	n := notify.Notification{
		Summary: "Meeting invitation: " + eventTitle(ev),
		Body:    invitationBody(ev, e.settings(), e.loc),
		Actions: []notify.Action{
			{Key: "accept", Label: "Accept"},
			{Key: "tentative", Label: "Maybe"},
			{Key: "decline", Label: "Decline"},
		},
		Resident: e.settings().ReminderPersist,
	}

	id, err := e.sender.Send(n)
	if err != nil {
		log.Warnf("invitations: send: %v", err)
		return
	}

	if err := e.repo.SetInvitationNotified(ctx, calID, ev.UID, ev.Start, now); err != nil {
		log.Warnf("invitations: record notified: %v", err)
	}

	e.mu.Lock()
	e.pending[id] = pendingInvite{eventID: ev.ID, calendarID: calID}
	e.mu.Unlock()
}

// HandleAction reacts to invitation notification buttons; wired to the notify
// client's dispatcher alongside the reminders engine, which is why it ignores
// notification ids it does not own.
func (e *Engine) HandleAction(id uint32, action string) {
	e.mu.Lock()
	inv, ok := e.pending[id]
	e.mu.Unlock()
	if !ok {
		return
	}

	switch action {
	case "accept", "decline", "tentative":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := rsvp.Apply(ctx, e.stores, inv.eventID, action)
		if err != nil {
			log.Warnf("invitations: respond %s: %v", action, err)
			return
		}
		if e.publish != nil {
			e.publish("events", map[string]any{"type": "changed", "calendarId": res.CalendarID})
		}
		e.sender.Dismiss(id)
	case "default":
		e.open()
		e.sender.Dismiss(id)
	}
}

func (e *Engine) HandleClosed(id uint32) {
	e.mu.Lock()
	delete(e.pending, id)
	e.mu.Unlock()
}

// domainSnapshot builds the minimal domain event SelfResponse needs from a
// stored row.
func domainSnapshot(ev *ent.Event) *calendar.Event {
	return &calendar.Event{
		Attendees: calendar.AttendeesFromMaps(ev.Attendees),
		Organizer: calendar.OrganizerFromMap(ev.Organizer),
	}
}

func eventCalendarID(ev *ent.Event) string {
	if ev.Edges.Calendar == nil {
		return ""
	}
	return ev.Edges.Calendar.ID
}

func eventTitle(ev *ent.Event) string {
	if ev.Summary == "" {
		return "(untitled event)"
	}
	return ev.Summary
}

func invitationBody(ev *ent.Event, s settings.UISettings, loc *time.Location) string {
	start := ev.Start.In(loc)
	var when string
	switch {
	case ev.AllDay:
		when = "All day " + start.Format("Mon, Jan 2")
	default:
		when = fmt.Sprintf("%s at %s", start.Format("Mon, Jan 2"), s.Clock(start))
	}

	lines := []string{when}
	if ev.Organizer != nil {
		switch org := calendar.AttendeeFromMap(ev.Organizer); {
		case org.DisplayName != "":
			lines = append(lines, "From "+org.DisplayName)
		case org.Email != "":
			lines = append(lines, "From "+org.Email)
		}
	}
	if ev.Location != "" {
		lines = append(lines, ev.Location)
	}
	return strings.Join(lines, "\n")
}

func openApp() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "show")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
}
