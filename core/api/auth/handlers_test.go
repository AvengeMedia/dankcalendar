package auth_handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auth_handler "github.com/AvengeMedia/dankcalendar/core/api/auth"
)

func TestCallbackHandler(t *testing.T) {
	t.Run("missing state", func(t *testing.T) {
		broker := auth_handler.NewCallbackBroker()
		rr := httptest.NewRecorder()

		auth_handler.NewCallbackHandler(broker)(rr, httptest.NewRequest(http.MethodGet, "/oauth/callback", nil))

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("unknown state", func(t *testing.T) {
		broker := auth_handler.NewCallbackBroker()
		rr := httptest.NewRecorder()

		auth_handler.NewCallbackHandler(broker)(rr, httptest.NewRequest(http.MethodGet, "/oauth/callback?state=nope", nil))

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("delivers payload to pending flow", func(t *testing.T) {
		broker := auth_handler.NewCallbackBroker()
		ch := broker.Register("st-1")
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodGet, "/oauth/callback?state=st-1&code=auth-code", nil)
		auth_handler.NewCallbackHandler(broker)(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Authorization received")

		payload, ok := <-ch
		require.True(t, ok)
		assert.Equal(t, "auth-code", payload.Code)
		assert.Equal(t, "st-1", payload.State)
		assert.Empty(t, payload.Error)
	})

	t.Run("propagates provider error", func(t *testing.T) {
		broker := auth_handler.NewCallbackBroker()
		ch := broker.Register("st-2")
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodGet, "/oauth/callback?state=st-2&error=access_denied", nil)
		auth_handler.NewCallbackHandler(broker)(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		payload, ok := <-ch
		require.True(t, ok)
		assert.Equal(t, "access_denied", payload.Error)
	})
}
