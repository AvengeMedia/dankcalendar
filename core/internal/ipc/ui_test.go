package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIOpenEventRequiresUID(t *testing.T) {
	deps := Deps{Bus: NewEventBus(), Pending: &PendingOpen{}}
	out := routeAndRead(t, Request{ID: 1, Method: "ui.openEvent", Params: map[string]any{"uid": "  "}}, deps)

	errMsg, ok := out["error"].(string)
	require.True(t, ok)
	assert.Contains(t, errMsg, "ui.openEvent requires a uid")
	assert.Nil(t, deps.Pending.Take())
}

func TestUIOpenEventStashesWithoutSubscriber(t *testing.T) {
	deps := Deps{Bus: NewEventBus(), Pending: &PendingOpen{}}
	out := routeAndRead(t, Request{ID: 2, Method: "ui.openEvent", Params: map[string]any{"uid": "evt-1", "start": "2026-07-01T09:00:00Z"}}, deps)

	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, result["ok"])

	assert.Equal(t, map[string]any{
		"action": "openEvent",
		"uid":    "evt-1",
		"start":  "2026-07-01T09:00:00Z",
	}, deps.Pending.Take())
}

func TestUIOpenEventPublishesToSubscriber(t *testing.T) {
	bus := NewEventBus()
	deps := Deps{Bus: bus, Pending: &PendingOpen{}}

	client, srv := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = srv.Close()
	})
	sub := bus.NewSubscriber(context.Background(), NewConnWriter(srv))
	sub.Subscribe("ui")
	t.Cleanup(sub.Close)

	events := make(chan map[string]any, 1)
	go func() {
		line, err := bufio.NewReader(client).ReadBytes('\n')
		if err != nil {
			return
		}
		var env map[string]any
		_ = json.Unmarshal(line, &env)
		events <- env
	}()

	out := routeAndRead(t, Request{ID: 3, Method: "ui.openEvent", Params: map[string]any{"uid": "evt-2"}}, deps)
	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, result["ok"])

	env := <-events
	assert.Equal(t, "ui", env["event"])
	assert.Equal(t, map[string]any{"action": "openEvent", "uid": "evt-2"}, env["data"])
	assert.Nil(t, deps.Pending.Take())
}

func TestUINewEventStashesWithoutSubscriber(t *testing.T) {
	deps := Deps{Bus: NewEventBus(), Pending: &PendingOpen{}}
	out := routeAndRead(t, Request{ID: 3, Method: "ui.newEvent", Params: map[string]any{"start": "2026-07-01T09:00:00Z"}}, deps)

	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, result["ok"])

	payload := deps.Pending.Take()
	require.NotNil(t, payload)
	assert.Equal(t, "newEvent", payload["action"])
	assert.Equal(t, "2026-07-01T09:00:00Z", payload["start"])
}
