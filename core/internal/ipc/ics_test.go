package ipc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/mocks"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

const inviteICS = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:x\r\nMETHOD:REQUEST\r\n" +
	"BEGIN:VEVENT\r\nUID:inv-1\r\nDTSTART:20260910T120000Z\r\nDTEND:20260910T130000Z\r\nSUMMARY:Planning\r\n" +
	"ORGANIZER:mailto:alice@example.com\r\nATTENDEE;PARTSTAT=NEEDS-ACTION:mailto:bob@example.com\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:inv-2\r\nDTSTART:20260911T120000Z\r\nDTEND:20260911T130000Z\r\nSUMMARY:Retro\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

type icsFixture struct {
	repo     *repo.Repo
	registry *calendar.Registry
	deps     Deps
}

func newIcsFixture(t *testing.T, kind account.Kind, readOnly bool) icsFixture {
	t.Helper()
	ctx := context.Background()
	client, err := repo.OpenMemory(ctx)
	require.NoError(t, err)
	r := repo.New(client)
	t.Cleanup(func() { _ = r.Close() })

	_, err = r.CreateAccount(ctx, repo.CreateAccountInput{ID: "acc", Kind: kind, DisplayName: "Acc"})
	require.NoError(t, err)
	_, err = r.UpsertCalendar(ctx, repo.UpsertCalendarInput{ID: "cal", AccountID: "acc", RemoteID: "remote", Name: "Work", ReadOnly: readOnly})
	require.NoError(t, err)

	registry := calendar.NewRegistry()
	return icsFixture{repo: r, registry: registry, deps: Deps{Repo: r, Registry: registry, Bus: NewEventBus()}}
}

func (f icsFixture) register(t *testing.T, kind calendar.AccountKind, provider calendar.Provider) {
	f.registry.Register(factoryFor(t, kind, provider))
}

func resultOf(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	require.Nil(t, out["error"], "unexpected error: %v", out["error"])
	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	return result
}

func resultEvents(t *testing.T, result map[string]any) []map[string]any {
	t.Helper()
	raw, ok := result["events"].([]any)
	require.True(t, ok)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		require.True(t, ok)
		out = append(out, entry)
	}
	return out
}

func TestEventsParseIcsMatchesSyncedEvents(t *testing.T) {
	f := newIcsFixture(t, account.KindCaldav, false)
	_, err := f.repo.UpsertEvent(context.Background(), repo.UpsertEventInput{
		CalendarID: "cal", UID: "inv-1", Summary: "Planning (synced)",
		Start: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC), End: time.Date(2026, 9, 10, 13, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	result := resultOf(t, routeAndRead(t, Request{ID: 1, Method: "events.parseIcs", Params: map[string]any{"ics": inviteICS}}, f.deps))
	assert.Equal(t, "REQUEST", result["method"])

	events := resultEvents(t, result)
	require.Len(t, events, 2)

	first := events[0]["event"].(map[string]any)
	assert.Equal(t, "inv-1", first["uid"])
	assert.Equal(t, "Planning", first["summary"])
	existing, ok := events[0]["existing"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Planning (synced)", existing["summary"])
	assert.Equal(t, "cal", existing["calendarId"])

	assert.Nil(t, events[1]["existing"])
}

func TestEventsParseIcsRejectsBadInput(t *testing.T) {
	f := newIcsFixture(t, account.KindCaldav, false)
	for name, params := range map[string]map[string]any{
		"missing": {},
		"garbage": {"ics": "not a calendar"},
	} {
		t.Run(name, func(t *testing.T) {
			out := routeAndRead(t, Request{ID: 1, Method: "events.parseIcs", Params: params}, f.deps)
			assert.NotEmpty(t, out["error"])
		})
	}
}

func TestEventsImportIcsCreatesAndSkipsExisting(t *testing.T) {
	f := newIcsFixture(t, account.KindCaldav, false)
	provider := mocks.NewMockProvider(t)
	provider.EXPECT().Close().Return(nil).Maybe()
	provider.EXPECT().CreateEvent(mock.Anything, mock.Anything, mock.MatchedBy(func(ev *calendar.Event) bool {
		return ev.UID == "inv-1" && ev.Organizer == nil && ev.Attendees == nil
	})).RunAndReturn(func(_ context.Context, _ calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
		out := *ev
		out.RemoteID = "remote/inv-1.ics"
		return &out, nil
	}).Once()
	f.register(t, calendar.AccountCalDAV, provider)

	params := map[string]any{"ics": inviteICS, "calendarId": "cal", "uids": []any{"inv-1"}}
	result := resultOf(t, routeAndRead(t, Request{ID: 1, Method: "events.importIcs", Params: params}, f.deps))
	assert.Equal(t, float64(1), result["imported"])
	events := resultEvents(t, result)
	require.Len(t, events, 1)
	assert.Equal(t, false, events[0]["existing"])

	stored, err := f.repo.FindEventByUID(context.Background(), "cal", "inv-1")
	require.NoError(t, err)
	assert.Equal(t, "Planning", stored.Summary)
	assert.Equal(t, "remote/inv-1.ics", stored.RemoteID)

	second := mocks.NewMockProvider(t)
	second.EXPECT().Close().Return(nil).Maybe()
	f.registry.Register(factoryFor(t, calendar.AccountCalDAV, second))

	result = resultOf(t, routeAndRead(t, Request{ID: 2, Method: "events.importIcs", Params: params}, f.deps))
	assert.Equal(t, float64(0), result["imported"])
	events = resultEvents(t, result)
	require.Len(t, events, 1)
	assert.Equal(t, true, events[0]["existing"])
}

type importingMock struct {
	*mocks.MockProvider
	got *calendar.Event
}

func (p *importingMock) ImportEvent(_ context.Context, _ calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	p.got = ev
	return ev, nil
}

func TestEventsImportIcsUsesImporterWithParticipation(t *testing.T) {
	f := newIcsFixture(t, account.KindGoogle, false)
	provider := &importingMock{MockProvider: mocks.NewMockProvider(t)}
	provider.EXPECT().Close().Return(nil).Maybe()
	f.register(t, calendar.AccountGoogle, provider)

	result := resultOf(t, routeAndRead(t, Request{ID: 1, Method: "events.importIcs", Params: map[string]any{"ics": inviteICS, "calendarId": "cal"}}, f.deps))
	assert.Equal(t, float64(2), result["imported"])

	stored, err := f.repo.FindEventByUID(context.Background(), "cal", "inv-1")
	require.NoError(t, err)
	require.Len(t, stored.Attendees, 1)
	assert.Equal(t, "bob@example.com", stored.Attendees[0]["email"])
	assert.Equal(t, "alice@example.com", stored.Organizer["email"])
}

func TestEventsImportIcsValidation(t *testing.T) {
	t.Run("read-only calendar", func(t *testing.T) {
		f := newIcsFixture(t, account.KindCaldav, true)
		out := routeAndRead(t, Request{ID: 1, Method: "events.importIcs", Params: map[string]any{"ics": inviteICS, "calendarId": "cal"}}, f.deps)
		assert.Contains(t, out["error"], "read-only")
	})

	t.Run("unknown uids", func(t *testing.T) {
		f := newIcsFixture(t, account.KindCaldav, false)
		out := routeAndRead(t, Request{ID: 1, Method: "events.importIcs", Params: map[string]any{"ics": inviteICS, "calendarId": "cal", "uids": []any{"nope"}}}, f.deps)
		assert.Contains(t, out["error"], "no matching events")
	})

	t.Run("missing calendar", func(t *testing.T) {
		f := newIcsFixture(t, account.KindCaldav, false)
		out := routeAndRead(t, Request{ID: 1, Method: "events.importIcs", Params: map[string]any{"ics": inviteICS}}, f.deps)
		assert.Contains(t, out["error"], "calendarId is required")
	})
}

func TestUIOpenIcsPublishesValidatedData(t *testing.T) {
	deps := Deps{Bus: NewEventBus(), Pending: &PendingOpen{}}

	out := routeAndRead(t, Request{ID: 1, Method: "ui.openIcs", Params: map[string]any{"ics": "garbage"}}, deps)
	assert.NotEmpty(t, out["error"])
	assert.Nil(t, deps.Pending.Take())

	resultOf(t, routeAndRead(t, Request{ID: 2, Method: "ui.openIcs", Params: map[string]any{"ics": inviteICS, "name": "invite.ics"}}, deps))
	assert.Equal(t, map[string]any{"action": "importIcs", "ics": inviteICS, "name": "invite.ics"}, deps.Pending.Take())
}

func factoryFor(t *testing.T, kind calendar.AccountKind, provider calendar.Provider) *mocks.MockProviderFactory {
	factory := mocks.NewMockProviderFactory(t)
	factory.EXPECT().Kind().Return(kind)
	factory.EXPECT().Build(mock.Anything, mock.Anything, mock.Anything).Return(provider, nil)
	return factory
}
