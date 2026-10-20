package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/duxlabs-dev/dux-calendar/core/internal/settings"
)

func TestUIEnv(t *testing.T) {
	tests := []struct {
		name             string
		scale            float64
		envScale         bool
		showAfterRestart bool
		want             []string
	}{
		{
			name:  "default scale leaves QT_SCALE_FACTOR alone",
			scale: 1,
			want:  []string{"DANKCAL_UI_SCALE=1"},
		},
		{
			name:  "custom scale sets QT_SCALE_FACTOR",
			scale: 1.25,
			want:  []string{"DANKCAL_UI_SCALE=1.25", "QT_SCALE_FACTOR=1.25"},
		},
		{
			name:             "restart from the UI clears start hidden",
			scale:            1,
			showAfterRestart: true,
			want:             []string{"DANKCAL_UI_SCALE=1", "DANKCAL_START_HIDDEN="},
		},
		{
			name:     "QT_SCALE_FACTOR from the environment wins",
			scale:    1.5,
			envScale: true,
		},
		{
			name:             "environment scale still reopens after restart",
			scale:            1.5,
			envScale:         true,
			showAfterRestart: true,
			want:             []string{"DANKCAL_START_HIDDEN="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := settings.Defaults()
			s.UIScale = tt.scale
			assert.Equal(t, tt.want, uiEnv(s, tt.envScale, tt.showAfterRestart))
		})
	}
}
