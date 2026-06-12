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

func routeAndRead(t *testing.T, req Request, deps Deps) map[string]any {
	t.Helper()
	client, srv := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = srv.Close()
	})

	go Route(context.Background(), newConnWriter(srv), req, deps, nil)

	line, err := bufio.NewReader(client).ReadBytes('\n')
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(line, &out))
	return out
}

func TestRoutePing(t *testing.T) {
	out := routeAndRead(t, Request{ID: 7, Method: "ping"}, Deps{})

	assert.Equal(t, float64(7), out["id"])
	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, result["pong"])
}

func TestRouteVersion(t *testing.T) {
	out := routeAndRead(t, Request{ID: 1, Method: "version"}, Deps{Version: "1.2.3"})

	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "1.2.3", result["version"])
	assert.Equal(t, float64(APIVersion), result["apiVersion"])
}

func TestRouteUnknownMethod(t *testing.T) {
	out := routeAndRead(t, Request{ID: 2, Method: "nope.nothing"}, Deps{})

	errMsg, ok := out["error"].(string)
	require.True(t, ok)
	assert.Contains(t, errMsg, "unknown method: nope.nothing")
	assert.Nil(t, out["result"])
}

func TestParamString(t *testing.T) {
	params := map[string]any{"name": "main", "count": 3}

	assert.Equal(t, "main", ParamString(params, "name"))
	assert.Empty(t, ParamString(params, "count"))
	assert.Empty(t, ParamString(params, "missing"))
}

func TestParamInt(t *testing.T) {
	params := map[string]any{"float": float64(5), "int": 7, "str": "9"}

	assert.Equal(t, 5, ParamInt(params, "float"))
	assert.Equal(t, 7, ParamInt(params, "int"))
	assert.Zero(t, ParamInt(params, "str"))
	assert.Zero(t, ParamInt(params, "missing"))
}

func TestParamBool(t *testing.T) {
	params := map[string]any{"bool": true, "str": "true", "bad": "nope", "num": 1}

	assert.True(t, ParamBool(params, "bool"))
	assert.True(t, ParamBool(params, "str"))
	assert.False(t, ParamBool(params, "bad"))
	assert.False(t, ParamBool(params, "num"))
	assert.False(t, ParamBool(params, "missing"))
}

func TestParamStringSlice(t *testing.T) {
	params := map[string]any{
		"ids":   []any{"a", "b", 3},
		"plain": "x",
	}

	assert.Equal(t, []string{"a", "b"}, ParamStringSlice(params, "ids"))
	assert.Nil(t, ParamStringSlice(params, "plain"))
	assert.Nil(t, ParamStringSlice(params, "missing"))
}
