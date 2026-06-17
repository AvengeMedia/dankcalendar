package sync

import (
	"testing"
	"time"
)

func fixedEngine(now time.Time) *Engine {
	e := NewEngine(nil, nil, nil, 5*time.Minute)
	e.now = func() time.Time { return now }
	return e
}

func TestScheduleHonorsRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := fixedEngine(now)

	e.schedule("acc", 30*time.Minute)

	if e.due("acc", now) {
		t.Error("account should not be due immediately after scheduling")
	}
	if e.due("acc", now.Add(29*time.Minute)) {
		t.Error("account should not be due before its deadline")
	}
	if !e.due("acc", now.Add(31*time.Minute)) {
		t.Error("account should be due after its deadline")
	}
}

func TestScheduleFloorsAtMinInterval(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := fixedEngine(now)

	e.schedule("acc", time.Second)

	if e.due("acc", now.Add(minInterval-time.Second)) {
		t.Error("tiny RetryAfter must be floored to minInterval")
	}
	if !e.due("acc", now.Add(minInterval+time.Second)) {
		t.Error("account should be due once minInterval has elapsed")
	}
}

func TestScheduleZeroUsesDefaultInterval(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := fixedEngine(now)

	e.schedule("acc", 0)

	if !e.due("acc", now.Add(5*time.Minute+time.Second)) {
		t.Error("zero RetryAfter should fall back to the engine interval")
	}
}

func TestUnknownAccountIsDue(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := fixedEngine(now)

	if !e.due("never-scheduled", now) {
		t.Error("an unscheduled account must be due")
	}
}

func TestUntilNextReturnsSoonest(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := fixedEngine(now)

	e.schedule("a", 30*time.Minute)
	e.schedule("b", 10*time.Minute)

	if d := e.untilNext(); d != 10*time.Minute {
		t.Errorf("untilNext = %v, want 10m", d)
	}
}

func TestUntilNextEmptyParksAtMaxWake(t *testing.T) {
	e := fixedEngine(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if d := e.untilNext(); d != maxWake {
		t.Errorf("untilNext = %v, want %v", d, maxWake)
	}
}

func TestUntilNextOverdueReturnsZero(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := fixedEngine(now)
	e.schedule("a", 30*time.Minute)

	e.now = func() time.Time { return now.Add(time.Hour) }
	if d := e.untilNext(); d != 0 {
		t.Errorf("untilNext = %v, want 0 for an overdue account", d)
	}
}
