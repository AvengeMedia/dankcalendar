package errdefs_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/errdefs"
)

func TestCustomErrorIsMatchesByType(t *testing.T) {
	err := errdefs.NewCustomError(errdefs.ErrTypeNotFound, "event not found")

	assert.ErrorIs(t, err, errdefs.ErrNotFound)
	assert.NotErrorIs(t, err, errdefs.ErrConflict)
	assert.NotErrorIs(t, err, errors.New("event not found"))
}

func TestCustomErrorMessage(t *testing.T) {
	err := errdefs.NewCustomError(errdefs.ErrTypeInvalidInput, "bad timestamp")
	assert.Equal(t, "ErrInvalidInput: bad timestamp", err.Error())
}

func TestErrorTypeString(t *testing.T) {
	assert.Equal(t, "ErrUnauthorized", errdefs.ErrTypeUnauthorized.String())
	assert.Equal(t, "ErrUnknown", errdefs.ErrorType(99).String())
}

func TestCustomErrorStatus(t *testing.T) {
	tests := []struct {
		errType errdefs.ErrorType
		want    int
	}{
		{errdefs.ErrTypeInvalidInput, http.StatusBadRequest},
		{errdefs.ErrTypeNotFound, http.StatusNotFound},
		{errdefs.ErrTypeConflict, http.StatusConflict},
		{errdefs.ErrTypeUnauthorized, http.StatusUnauthorized},
		{errdefs.ErrTypeUnsupported, http.StatusNotImplemented},
		{errdefs.ErrTypeProvider, http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.errType.String(), func(t *testing.T) {
			assert.Equal(t, tc.want, errdefs.NewCustomError(tc.errType, "x").GetStatus())
		})
	}
}

func TestHumaErrorFunc(t *testing.T) {
	err := errdefs.HumaErrorFunc(http.StatusUnprocessableEntity, "validation failed",
		errors.New("first"), errors.New("second"))

	assert.Equal(t, http.StatusUnprocessableEntity, err.GetStatus())
	assert.Equal(t, "validation failed", err.Error())

	respErr, ok := err.(*errdefs.ResponseError)
	require.True(t, ok)
	assert.Equal(t, []string{"first", "second"}, respErr.Details)
}
