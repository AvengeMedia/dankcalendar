package ipc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/AvengeMedia/dankcalendar/core/ent"
)

func TestMapEventVisibility(t *testing.T) {
	tests := []struct {
		name       string
		visibility string
	}{
		{name: "private", visibility: "private"},
		{name: "confidential", visibility: "confidential"},
		{name: "unset", visibility: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := mapEvent(&ent.Event{UID: "abc", Visibility: tt.visibility})
			assert.Equal(t, tt.visibility, entry["visibility"])
		})
	}
}
