// Package reminders schedules desktop notifications for upcoming events.
// It expands events over a lookahead window, derives trigger times from each
// event's reminders (or the configured default), and fires each trigger
// exactly once per occurrence, tracked via ReminderState rows. Evaluation is
// event-driven: the engine parks until the next trigger is due and wakes
// early when calendar data changes or the system resumes from suspend.
package reminders

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	entevent "github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/notify"
	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

const (
	// lookahead matches the longest reminder offset providers allow (Google
	// caps overrides at four weeks).
	lookahead = 28 * 24 * time.Hour
	// startGrace still fires a missed reminder shortly after the event began,
	// e.g. right after the daemon restarts.
	startGrace = 5 * time.Minute
	// allDayLate bounds how stale an all-day trigger may be: those triggers
	// can sit days inside a multi-day occurrence, so the start-based grace
	// does not apply.
	allDayLate = 24 * time.Hour
	stateFloor = 35 * 24 * time.Hour
	pruneAfter = 60 * 24 * time.Hour
	// defaultMaxWake caps how long the engine parks; a backstop against a
	// missed wake signal or an undetected clock change, not the normal path.
	defaultMaxWake = time.Hour
)

type Sender interface {
	Send(n notify.Notification) (uint32, error)
	Dismiss(id uint32)
	OpenURI(uri string) error
}

type Publisher func(topic string, data any)

type stateKey struct {
	calendarID string
	uid        string
	start      time.Time
	minutes    int
}

type trigger struct {
	at      time.Time
	minutes int
}

type Upcoming struct {
	EventID    string    `json:"eventId,omitempty"`
	CalendarID string    `json:"calendarId"`
	UID        string    `json:"uid"`
	Summary    string    `json:"summary"`
	Start      time.Time `json:"start"`
	AllDay     bool      `json:"allDay"`
	Trigger    time.Time `json:"trigger"`
	Minutes    int       `json:"minutes"`
	Snoozed    bool      `json:"snoozed,omitempty"`
}

type Engine struct {
	repo     *repo.Repo
	sender   Sender
	publish  Publisher
	settings func() settings.UISettings
	now      func() time.Time
	loc      *time.Location
	open     func()
	maxWake  time.Duration

	wake chan struct{}

	mu      sync.Mutex
	pending map[uint32]firedReminder
	running bool
	stop    chan struct{}
}

// firedReminder records what a live notification needs once the user acts on it:
// the state key to snooze/clear and the meeting link to join.
type firedReminder struct {
	key        stateKey
	meetingURL string
}

func NewEngine(r *repo.Repo, sender Sender, maxWake time.Duration) *Engine {
	if maxWake <= 0 {
		maxWake = defaultMaxWake
	}
	return &Engine{
		repo:     r,
		sender:   sender,
		settings: settings.Load,
		now:      time.Now,
		loc:      time.Local,
		open:     openApp,
		maxWake:  maxWake,
		wake:     make(chan struct{}, 1),
		pending:  make(map[uint32]firedReminder),
		stop:     make(chan struct{}),
	}
}

func (e *Engine) SetPublisher(p Publisher) { e.publish = p }

func (e *Engine) Start(ctx context.Context) {
	if e.sender == nil {
		log.Warnf("reminders: no notification transport, engine not started")
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
	go e.watchSuspend(ctx)
}

// Wake triggers a reschedule; concurrent calls coalesce.
func (e *Engine) Wake() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// WatchMutations reschedules when calendar data changes. ReminderState is
// excluded: the engine writes it when firing, so hooking it would loop.
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
	if n, err := e.repo.PruneReminderStates(ctx, e.now().Add(-pruneAfter)); err == nil && n > 0 {
		log.Debugf("reminders: pruned %d stale states", n)
	}

	// NewTimer(0) evaluates immediately, then loop() parks it on the next due trigger.
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stop:
			return
		case <-e.wake:
		case <-timer.C:
		}

		if err := e.Tick(ctx); err != nil {
			log.Warnf("reminders: %v", err)
		}
		resetTimer(timer, e.untilNext(ctx))
	}
}

// untilNext is the delay until the next due reminder, capped at maxWake.
func (e *Engine) untilNext(ctx context.Context) time.Duration {
	if !e.settings().RemindersEnabled {
		return e.maxWake
	}
	up, err := e.Upcoming(ctx, 1)
	if err != nil {
		log.Warnf("reminders: schedule: %v", err)
		return e.maxWake
	}
	if len(up) == 0 {
		return e.maxWake
	}
	switch d := up[0].Trigger.Sub(e.now()); {
	case d < 0:
		return 0
	case d > e.maxWake:
		return e.maxWake
	default:
		return d
	}
}

// resetTimer rearms t, draining a pending fire so the next select sees only
// the new deadline.
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

func (e *Engine) Tick(ctx context.Context) error {
	s := e.settings()
	if !s.RemindersEnabled {
		return nil
	}

	now := e.now()
	events, resolved, err := e.candidates(ctx, now, s)
	if err != nil {
		return err
	}

	states, err := e.stateMap(ctx, now)
	if err != nil {
		return err
	}

	for _, ev := range events {
		calID := eventCalendarID(ev)
		if calID == "" {
			continue
		}
		cs := resolvedFor(resolved, calID, s)
		if !cs.RemindersEnabled {
			continue
		}
		for _, tr := range triggersFor(ev, cs, e.loc) {
			key := stateKey{calendarID: calID, uid: ev.UID, start: ev.Start.UTC(), minutes: tr.minutes}
			st, seen := states[key]
			switch {
			case !seen:
				if !due(tr, ev, now) {
					continue
				}
			case st.SnoozedUntil == nil:
				continue
			case st.SnoozedUntil.After(now):
				continue
			case !ev.End.After(now):
				continue
			}
			e.fire(ctx, ev, key, cs, now)
		}
	}
	return nil
}

// Upcoming returns the next pending triggers, soonest first. Snoozed
// reminders appear at their snooze deadline.
func (e *Engine) Upcoming(ctx context.Context, limit int) ([]Upcoming, error) {
	s := e.settings()
	now := e.now()

	events, resolved, err := e.candidates(ctx, now, s)
	if err != nil {
		return nil, err
	}
	states, err := e.stateMap(ctx, now)
	if err != nil {
		return nil, err
	}

	var out []Upcoming
	for _, ev := range events {
		calID := eventCalendarID(ev)
		if calID == "" {
			continue
		}
		cs := resolvedFor(resolved, calID, s)
		if !cs.RemindersEnabled {
			continue
		}
		for _, tr := range triggersFor(ev, cs, e.loc) {
			key := stateKey{calendarID: calID, uid: ev.UID, start: ev.Start.UTC(), minutes: tr.minutes}
			entry := Upcoming{
				EventID:    ev.ID,
				CalendarID: calID,
				UID:        ev.UID,
				Summary:    ev.Summary,
				Start:      ev.Start,
				AllDay:     ev.AllDay,
				Trigger:    tr.at,
				Minutes:    tr.minutes,
			}
			switch st, seen := states[key]; {
			case seen && st.SnoozedUntil != nil && st.SnoozedUntil.After(now):
				entry.Trigger = *st.SnoozedUntil
				entry.Snoozed = true
			case seen:
				continue
			case !tr.at.After(now):
				continue
			}
			out = append(out, entry)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Trigger.Before(out[j].Trigger) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// SendTest fires an immediate notification so users can verify their setup.
func (e *Engine) SendTest() error {
	if e.sender == nil {
		return fmt.Errorf("desktop notifications unavailable (no session bus)")
	}
	_, err := e.sender.Send(notify.Notification{
		Summary:  "Dank Calendar",
		Body:     "Test reminder — notifications are working.",
		Resident: e.settings().ReminderPersist,
	})
	return err
}

// HandleAction reacts to notification buttons; wired to the notify client's
// signal dispatcher.
func (e *Engine) HandleAction(id uint32, action string) {
	e.mu.Lock()
	fired, ok := e.pending[id]
	e.mu.Unlock()
	if !ok {
		return
	}
	key := fired.key

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch action {
	case "snooze":
		s := e.settings()
		if cal, err := e.repo.GetCalendar(ctx, key.calendarID); err == nil {
			s = cal.ReminderOverrides.Resolve(s)
		}
		until := e.now().Add(time.Duration(s.SnoozeMinutes) * time.Minute)
		if err := e.repo.SnoozeReminder(ctx, key.calendarID, key.uid, key.start, key.minutes, until); err != nil {
			log.Warnf("reminders: snooze: %v", err)
		}
		e.sender.Dismiss(id)
	case "join":
		if err := e.sender.OpenURI(fired.meetingURL); err != nil {
			log.Warnf("reminders: open meeting: %v", err)
		}
		e.sender.Dismiss(id)
	case "default":
		e.open()
		e.sender.Dismiss(id)
	case "dismiss":
		e.sender.Dismiss(id)
	}
}

func (e *Engine) HandleClosed(id uint32) {
	e.mu.Lock()
	delete(e.pending, id)
	e.mu.Unlock()
}

// candidates returns non-cancelled occurrences from visible calendars that
// have not ended, expanded over the lookahead window. The window starts in
// the recent past so occurrences that just began still expand.
func (e *Engine) candidates(ctx context.Context, now time.Time, base settings.UISettings) ([]*ent.Event, map[string]settings.UISettings, error) {
	cals, err := e.repo.ListCalendars(ctx)
	if err != nil {
		return nil, nil, err
	}
	hidden := make(map[string]struct{})
	resolved := make(map[string]settings.UISettings, len(cals))
	for _, c := range cals {
		if c.Hidden {
			hidden[c.ID] = struct{}{}
		}
		resolved[c.ID] = c.ReminderOverrides.Resolve(base)
	}

	from := now.Add(-(allDayLate + 2*time.Hour))
	to := now.Add(lookahead)
	events, _, err := e.repo.ListEvents(ctx, repo.ListEventsParams{
		Filter: repo.EventFilter{From: &from, To: &to, IncludeRecurring: true},
	})
	if err != nil {
		return nil, nil, err
	}

	out := events[:0]
	for _, ev := range events {
		if ev.Status == entevent.StatusCancelled {
			continue
		}
		if _, ok := hidden[eventCalendarID(ev)]; ok {
			continue
		}
		out = append(out, ev)
	}
	return out, resolved, nil
}

// resolvedFor returns the calendar's resolved settings, falling back to the
// global base when the calendar is unknown.
func resolvedFor(resolved map[string]settings.UISettings, calID string, base settings.UISettings) settings.UISettings {
	if cs, ok := resolved[calID]; ok {
		return cs
	}
	return base
}

func (e *Engine) stateMap(ctx context.Context, now time.Time) (map[stateKey]*ent.ReminderState, error) {
	states, err := e.repo.ListReminderStates(ctx, now.Add(-stateFloor), now.Add(lookahead))
	if err != nil {
		return nil, err
	}
	out := make(map[stateKey]*ent.ReminderState, len(states))
	for _, st := range states {
		out[stateKey{calendarID: st.CalendarID, uid: st.UID, start: st.OccurrenceStart.UTC(), minutes: st.Minutes}] = st
	}
	return out, nil
}

func (e *Engine) fire(ctx context.Context, ev *ent.Event, key stateKey, s settings.UISettings, now time.Time) {
	actions := make([]notify.Action, 0, 4)
	if ev.MeetingURL != "" {
		actions = append(actions, notify.Action{Key: "join", Label: "Join"})
	}
	actions = append(actions,
		notify.Action{Key: "default", Label: "Open"},
		notify.Action{Key: "snooze", Label: fmt.Sprintf("Snooze %d min", s.SnoozeMinutes)},
		notify.Action{Key: "dismiss", Label: "Dismiss"},
	)

	n := notify.Notification{
		Summary:  eventTitle(ev),
		Body:     eventBody(ev, now, s, e.loc),
		Actions:  actions,
		Resident: s.ReminderPersist,
	}

	id, err := e.sender.Send(n)
	if err != nil {
		log.Warnf("reminders: send: %v", err)
		return
	}

	if err := e.repo.SetReminderFired(ctx, key.calendarID, key.uid, key.start, key.minutes, now); err != nil {
		log.Warnf("reminders: record fired: %v", err)
	}

	e.mu.Lock()
	e.pending[id] = firedReminder{key: key, meetingURL: ev.MeetingURL}
	e.mu.Unlock()

	if e.publish != nil {
		e.publish("reminders", map[string]any{
			"type":       "fired",
			"eventId":    ev.ID,
			"calendarId": key.calendarID,
			"uid":        key.uid,
			"summary":    ev.Summary,
			"start":      ev.Start,
			"minutes":    key.minutes,
		})
	}
}

func due(tr trigger, ev *ent.Event, now time.Time) bool {
	if tr.at.After(now) {
		return false
	}
	switch {
	case ev.AllDay:
		return tr.at.After(now.Add(-allDayLate)) && ev.End.After(now)
	default:
		return ev.Start.After(now.Add(-startGrace))
	}
}

// triggersFor derives the trigger times for one occurrence. Explicit popup
// reminders win; otherwise the configured defaults apply. All-day triggers
// are computed against local midnight of the start date so "09:00 the day
// before" means wall-clock time, not UTC.
func triggersFor(ev *ent.Event, s settings.UISettings, loc *time.Location) []trigger {
	if ev.AllDay && !s.AllDayReminders {
		return nil
	}

	explicit := popupMinutes(ev.Reminders)
	base := ev.Start
	if ev.AllDay {
		base = localMidnight(ev.Start, loc)
	}

	if len(explicit) == 0 {
		switch {
		case ev.AllDay:
			hour, minute := s.AllDayClock()
			day := base.AddDate(0, 0, -s.AllDayReminderDaysBefore)
			at := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
			return []trigger{{at: at, minutes: int(base.Sub(at).Minutes())}}
		case s.DefaultReminderMinutes < 0:
			return nil
		default:
			minutes := s.DefaultReminderMinutes
			return []trigger{{at: base.Add(-time.Duration(minutes) * time.Minute), minutes: minutes}}
		}
	}

	out := make([]trigger, 0, len(explicit))
	for _, minutes := range explicit {
		out = append(out, trigger{at: base.Add(-time.Duration(minutes) * time.Minute), minutes: minutes})
	}
	return out
}

// popupMinutes extracts offsets for display-style reminders, dropping email
// reminders and duplicates.
func popupMinutes(reminders []map[string]any) []int {
	var out []int
	seen := make(map[int]struct{})
	for _, rem := range calendar.RemindersFromMaps(reminders) {
		switch rem.Method {
		case "", "popup", "display":
		default:
			continue
		}
		if _, dup := seen[rem.Minutes]; dup {
			continue
		}
		seen[rem.Minutes] = struct{}{}
		out = append(out, rem.Minutes)
	}
	return out
}

func localMidnight(start time.Time, loc *time.Location) time.Time {
	y, m, d := start.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
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

func eventBody(ev *ent.Event, now time.Time, s settings.UISettings, loc *time.Location) string {
	var when string
	switch {
	case ev.AllDay:
		when = "All day " + dayPhrase(localMidnight(ev.Start, loc), now.In(loc))
	default:
		start := ev.Start.In(loc)
		verb := "Starts"
		if start.Before(now) {
			verb = "Started"
		}
		when = fmt.Sprintf("%s %s at %s", verb, dayPhrase(start, now.In(loc)), s.Clock(start))
	}

	if ev.Location != "" {
		return when + "\n" + ev.Location
	}
	return when
}

func dayPhrase(t, now time.Time) string {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Round absorbs DST-shortened or -lengthened days.
	switch int(math.Round(day.Sub(today).Hours() / 24)) {
	case 0:
		return "today"
	case 1:
		return "tomorrow"
	case -1:
		return "yesterday"
	default:
		return "on " + t.Format("Mon, Jan 2")
	}
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
