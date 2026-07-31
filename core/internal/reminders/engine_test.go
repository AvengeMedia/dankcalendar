package reminders

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/ent/event"
	"github.com/AvengeMedia/dankcalendar/core/ent/task"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
	"github.com/AvengeMedia/dankcalendar/core/internal/notify"
	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

var t0 = time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)

type EngineSuite struct {
	suite.Suite
	ctx    context.Context
	repo   *repo.Repo
	sender *mocks.MockSender
	engine *Engine
	cfg    settings.UISettings
	clock  time.Time
}

func TestEngineSuite(t *testing.T) {
	suite.Run(t, new(EngineSuite))
}

func (s *EngineSuite) SetupTest() {
	s.ctx = context.Background()

	client, err := repo.OpenMemory(s.ctx)
	s.Require().NoError(err)
	s.repo = repo.New(client)
	s.T().Cleanup(func() { _ = s.repo.Close() })

	_, err = s.repo.CreateAccount(s.ctx, repo.CreateAccountInput{
		ID:          "acc",
		Kind:        account.KindLocal,
		DisplayName: "Acc",
	})
	s.Require().NoError(err)

	_, err = s.repo.UpsertCalendar(s.ctx, repo.UpsertCalendarInput{
		ID:        "cal",
		AccountID: "acc",
		RemoteID:  "remote-cal",
		Name:      "Calendar",
	})
	s.Require().NoError(err)

	s.cfg = settings.Defaults()
	s.clock = t0

	s.sender = mocks.NewMockSender(s.T())
	s.engine = NewEngine(s.repo, s.sender, time.Minute)
	s.engine.settings = func() settings.UISettings { return s.cfg }
	s.engine.now = func() time.Time { return s.clock }
	s.engine.loc = time.UTC
	s.engine.open = func() {}
}

func (s *EngineSuite) addEvent(uid string, start, end time.Time, mutate ...func(*repo.UpsertEventInput)) *ent.Event {
	in := repo.UpsertEventInput{
		CalendarID: "cal",
		UID:        uid,
		Summary:    "Event " + uid,
		Status:     event.StatusConfirmed,
		Start:      start,
		End:        end,
	}
	for _, m := range mutate {
		m(&in)
	}
	ev, err := s.repo.UpsertEvent(s.ctx, in)
	s.Require().NoError(err)
	return ev
}

func (s *EngineSuite) addTask(uid string, mutate ...func(*repo.UpsertTaskInput)) *ent.Task {
	in := repo.UpsertTaskInput{
		CalendarID: "cal",
		UID:        uid,
		Summary:    "Task " + uid,
		Status:     task.StatusNeedsAction,
	}
	for _, m := range mutate {
		m(&in)
	}
	got, err := s.repo.UpsertTask(s.ctx, in)
	s.Require().NoError(err)
	return got
}

func (s *EngineSuite) expectSend(id uint32) *notify.Notification {
	captured := &notify.Notification{}
	s.sender.EXPECT().Send(mock.Anything).Run(func(n notify.Notification) {
		*captured = n
	}).Return(id, nil).Once()
	return captured
}

func (s *EngineSuite) TestDefaultReminderFiresOnce() {
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Event ev1", captured.Summary)
	s.Equal("Starts today at 12:10", captured.Body)

	// Already recorded as fired: no second notification.
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestHighPriorityTaskDueUsesCriticalAlarm() {
	s.cfg.NotificationSounds = true
	s.addTask("task1", func(in *repo.UpsertTaskInput) {
		in.Due = t0
		in.Priority = 2
	})
	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Task task1", captured.Summary)
	s.Equal("Due today at 12:00", captured.Body)
	s.Equal(notify.UrgencyCritical, captured.Urgency)
	s.Equal(notify.SoundTask, captured.SoundName)

	// The reminder state prevents a second notification for the same task time.
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestTaskStartAndDueBothNotifyWithMappedPriority() {
	s.addTask("task1", func(in *repo.UpsertTaskInput) {
		in.Start = t0
		in.Due = t0.Add(2 * time.Minute)
		in.Priority = 8
	})
	started := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Starts today at 12:00", started.Body)
	s.Equal(notify.UrgencyLow, started.Urgency)

	s.clock = t0.Add(2 * time.Minute)
	due := s.expectSend(2)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Due today at 12:02", due.Body)
}

func (s *EngineSuite) TestCompletedTaskDoesNotNotify() {
	s.addTask("task1", func(in *repo.UpsertTaskInput) {
		in.Due = t0
		in.Status = task.StatusCompleted
	})
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestExplicitReminderWinsAndEmailSkipped() {
	s.addEvent("ev1", t0.Add(5*time.Minute), t0.Add(time.Hour), func(in *repo.UpsertEventInput) {
		in.Reminders = []map[string]any{
			{"method": "popup", "minutes": 5},
			{"method": "email", "minutes": 60},
		}
	})

	s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestRemindersDisabled() {
	s.cfg.RemindersEnabled = false
	s.addEvent("ev1", t0.Add(5*time.Minute), t0.Add(time.Hour))

	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestNoDefaultReminder() {
	s.cfg.DefaultReminderMinutes = -1
	s.addEvent("ev1", t0.Add(5*time.Minute), t0.Add(time.Hour))

	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestFutureTriggerNotDue() {
	s.addEvent("ev1", t0.Add(2*time.Hour), t0.Add(3*time.Hour))

	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestUntilNextSchedulesToTrigger() {
	s.engine.maxWake = time.Hour
	// Default 10m reminder on an event 30m out → trigger 20m from now.
	s.addEvent("ev1", t0.Add(30*time.Minute), t0.Add(time.Hour))

	s.Equal(20*time.Minute, s.engine.untilNext(s.ctx))
}

func (s *EngineSuite) TestUntilNextCapsAtMaxWake() {
	s.engine.maxWake = time.Minute
	s.addEvent("ev1", t0.Add(30*time.Minute), t0.Add(time.Hour))

	s.Equal(time.Minute, s.engine.untilNext(s.ctx))
}

func (s *EngineSuite) TestUntilNextBackstopWhenIdle() {
	s.engine.maxWake = time.Hour

	s.Equal(time.Hour, s.engine.untilNext(s.ctx))
}

func (s *EngineSuite) TestStaleTimedReminderSkipped() {
	s.addEvent("ev1", t0.Add(-30*time.Minute), t0.Add(time.Hour))

	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestTwelveHourClockInBody() {
	s.cfg.Use24HourClock = false
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Starts today at 12:10 PM", captured.Body)
}

func (s *EngineSuite) TestJustStartedStillFires() {
	s.addEvent("ev1", t0.Add(-2*time.Minute), t0.Add(time.Hour))

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Started today at 11:58", captured.Body)
}

func (s *EngineSuite) TestAllDayGatedBySetting() {
	dayStart := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	s.addEvent("ev1", dayStart, dayStart.AddDate(0, 0, 1), func(in *repo.UpsertEventInput) {
		in.AllDay = true
	})

	// Disabled by default: nothing fires even though 09:00 has passed.
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.cfg.AllDayReminders = true
	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("All day today", captured.Body)
}

func (s *EngineSuite) TestSnoozeRefires() {
	s.cfg.SnoozeMinutes = 5
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	s.expectSend(7)
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.sender.EXPECT().Dismiss(uint32(7)).Once()
	s.engine.HandleAction(7, "snooze")

	// Not yet due again.
	s.clock = t0.Add(3 * time.Minute)
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.clock = t0.Add(6 * time.Minute)
	s.expectSend(8)
	s.Require().NoError(s.engine.Tick(s.ctx))

	// Refire cleared the snooze: no further notifications.
	s.clock = t0.Add(8 * time.Minute)
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestSnoozeUntilStartRefiresAtStart() {
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	s.expectSend(7)
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.sender.EXPECT().Dismiss(uint32(7)).Once()
	s.engine.HandleAction(7, "snooze_start")

	// Not yet due again.
	s.clock = t0.Add(5 * time.Minute)
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.clock = t0.Add(10 * time.Minute)
	captured := s.expectSend(8)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Starts today at 12:10", captured.Body)

	// Refire cleared the snooze: no further notifications.
	s.clock = t0.Add(12 * time.Minute)
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestSnoozeUntilStartAbsentOnceStarted() {
	s.addEvent("ev1", t0.Add(-2*time.Minute), t0.Add(time.Hour))

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	for _, a := range captured.Actions {
		s.NotEqual("snooze_start", a.Key)
	}
}

func (s *EngineSuite) TestSnoozeUntilStartAbsentForAllDay() {
	s.cfg.AllDayReminders = true
	s.cfg.AllDayReminderDaysBefore = 1
	s.cfg.AllDayReminderTime = "18:00"
	dayStart := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	s.addEvent("ev1", dayStart, dayStart.AddDate(0, 0, 1), func(in *repo.UpsertEventInput) {
		in.AllDay = true
	})

	// Past the 18:00 day-before trigger, before the event starts.
	s.clock = time.Date(2026, 6, 12, 18, 30, 0, 0, time.UTC)
	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	for _, a := range captured.Actions {
		s.NotEqual("snooze_start", a.Key)
	}
}

func (s *EngineSuite) TestSnoozeUntilStartAfterStartOnlyDismisses() {
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	s.expectSend(7)
	s.Require().NoError(s.engine.Tick(s.ctx))

	// The event began while the notification sat on screen: no snooze row is
	// written, so nothing refires.
	s.clock = t0.Add(11 * time.Minute)
	s.sender.EXPECT().Dismiss(uint32(7)).Once()
	s.engine.HandleAction(7, "snooze_start")
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestJoinActionOpensMeetingURL() {
	const url = "https://meet.google.com/abc-defg-hij"
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute), func(in *repo.UpsertEventInput) {
		in.MeetingURL = url
	})

	captured := s.expectSend(9)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Require().NotEmpty(captured.Actions)
	s.Equal("join", captured.Actions[0].Key)

	s.sender.EXPECT().OpenURI(url).Return(nil).Once()
	s.sender.EXPECT().Dismiss(uint32(9)).Once()
	s.engine.HandleAction(9, "join")
}

func (s *EngineSuite) TestNoJoinActionWithoutMeetingURL() {
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	for _, a := range captured.Actions {
		s.NotEqual("join", a.Key)
	}
}

func (s *EngineSuite) TestDismissedActionDoesNotRefire() {
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	s.expectSend(7)
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.sender.EXPECT().Dismiss(uint32(7)).Once()
	s.engine.HandleAction(7, "dismiss")
	s.engine.HandleClosed(7)

	s.clock = t0.Add(2 * time.Minute)
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestHiddenCalendarSkipped() {
	_, err := s.repo.UpsertCalendar(s.ctx, repo.UpsertCalendarInput{
		ID:        "cal2",
		AccountID: "acc",
		RemoteID:  "remote-cal2",
		Name:      "Hidden",
		Hidden:    true,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.repo.SetCalendarHidden(s.ctx, "cal2", true))

	in := repo.UpsertEventInput{
		CalendarID: "cal2",
		UID:        "ev1",
		Summary:    "Hidden event",
		Status:     event.StatusConfirmed,
		Start:      t0.Add(5 * time.Minute),
		End:        t0.Add(time.Hour),
	}
	_, err = s.repo.UpsertEvent(s.ctx, in)
	s.Require().NoError(err)

	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestCancelledEventSkipped() {
	s.addEvent("ev1", t0.Add(5*time.Minute), t0.Add(time.Hour), func(in *repo.UpsertEventInput) {
		in.Status = event.StatusCancelled
	})

	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestRecurringOccurrenceFires() {
	start := time.Date(2026, 6, 2, 12, 10, 0, 0, time.UTC)
	s.addEvent("ev1", start, start.Add(30*time.Minute), func(in *repo.UpsertEventInput) {
		in.Recurrence = map[string]any{"rrule": []string{"FREQ=DAILY"}}
	})

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.Equal("Starts today at 12:10", captured.Body)

	// Same occurrence stays deduped; the next day's occurrence fires fresh.
	s.Require().NoError(s.engine.Tick(s.ctx))

	s.clock = t0.AddDate(0, 0, 1)
	s.expectSend(2)
	s.Require().NoError(s.engine.Tick(s.ctx))
}

func (s *EngineSuite) TestUpcomingListsFutureTriggers() {
	s.addEvent("soon", t0.Add(2*time.Hour), t0.Add(3*time.Hour))
	s.addEvent("later", t0.Add(5*time.Hour), t0.Add(6*time.Hour))

	upcoming, err := s.engine.Upcoming(s.ctx, 0)
	s.Require().NoError(err)
	s.Require().Len(upcoming, 2)
	s.Equal("Event soon", upcoming[0].Summary)
	s.Equal(t0.Add(110*time.Minute), upcoming[0].Trigger.UTC())
	s.Equal("Event later", upcoming[1].Summary)
	s.False(upcoming[0].Snoozed)
}

func (s *EngineSuite) TestUpcomingShowsSnoozed() {
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute))

	s.expectSend(7)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.sender.EXPECT().Dismiss(uint32(7)).Once()
	s.engine.HandleAction(7, "snooze")

	upcoming, err := s.engine.Upcoming(s.ctx, 0)
	s.Require().NoError(err)
	s.Require().Len(upcoming, 1)
	s.True(upcoming[0].Snoozed)
	s.Equal(t0.Add(time.Duration(s.cfg.SnoozeMinutes)*time.Minute), upcoming[0].Trigger.UTC())
}

func (s *EngineSuite) TestNotificationCarriesSettings() {
	s.cfg.ReminderPersist = false
	s.cfg.SnoozeMinutes = 15
	s.addEvent("ev1", t0.Add(10*time.Minute), t0.Add(40*time.Minute), func(in *repo.UpsertEventInput) {
		in.Location = "Conference Room"
	})

	captured := s.expectSend(1)
	s.Require().NoError(s.engine.Tick(s.ctx))
	s.False(captured.Resident)
	s.Equal("Starts today at 12:10\nConference Room", captured.Body)
	s.Require().Len(captured.Actions, 4)
	s.Equal("Snooze 15 min", captured.Actions[1].Label)
	s.Equal("snooze_start", captured.Actions[2].Key)
}

func TestTriggersFor(t *testing.T) {
	cfg := settings.Defaults()
	cfg.AllDayReminders = true

	dayBefore := cfg
	dayBefore.AllDayReminderDaysBefore = 1
	dayBefore.AllDayReminderTime = "18:00"

	noDefault := cfg
	noDefault.DefaultReminderMinutes = -1

	allDayOff := settings.Defaults()

	timed := &ent.Event{
		Start: time.Date(2026, 6, 12, 15, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 6, 12, 16, 0, 0, 0, time.UTC),
	}
	allDay := &ent.Event{
		Start:  time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		End:    time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC),
		AllDay: true,
	}

	withReminders := func(ev *ent.Event, reminders []map[string]any) *ent.Event {
		out := *ev
		out.Reminders = reminders
		return &out
	}

	tests := []struct {
		name string
		ev   *ent.Event
		cfg  settings.UISettings
		want []trigger
	}{
		{
			name: "timed default",
			ev:   timed,
			cfg:  cfg,
			want: []trigger{{at: time.Date(2026, 6, 12, 14, 50, 0, 0, time.UTC), minutes: 10}},
		},
		{
			name: "timed no default",
			ev:   timed,
			cfg:  noDefault,
			want: nil,
		},
		{
			name: "timed explicit dedups and skips email",
			ev: withReminders(timed, []map[string]any{
				{"method": "popup", "minutes": 30},
				{"method": "popup", "minutes": float64(30)},
				{"method": "email", "minutes": 5},
			}),
			cfg:  cfg,
			want: []trigger{{at: time.Date(2026, 6, 12, 14, 30, 0, 0, time.UTC), minutes: 30}},
		},
		{
			name: "all day disabled",
			ev:   allDay,
			cfg:  allDayOff,
			want: nil,
		},
		{
			name: "all day default same-day time",
			ev:   allDay,
			cfg:  cfg,
			want: []trigger{{at: time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC), minutes: -540}},
		},
		{
			name: "all day default day before",
			ev:   allDay,
			cfg:  dayBefore,
			want: []trigger{{at: time.Date(2026, 6, 11, 18, 0, 0, 0, time.UTC), minutes: 360}},
		},
		{
			name: "all day explicit minutes from midnight",
			ev: withReminders(allDay, []map[string]any{
				{"method": "popup", "minutes": 600},
			}),
			cfg:  cfg,
			want: []trigger{{at: time.Date(2026, 6, 11, 14, 0, 0, 0, time.UTC), minutes: 600}},
		},
		{
			name: "all day disabled still fires explicit alarm",
			ev: withReminders(allDay, []map[string]any{
				{"method": "popup", "minutes": 600},
			}),
			cfg:  allDayOff,
			want: []trigger{{at: time.Date(2026, 6, 11, 14, 0, 0, 0, time.UTC), minutes: 600}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, triggersFor(tt.ev, tt.cfg, time.UTC))
		})
	}
}
