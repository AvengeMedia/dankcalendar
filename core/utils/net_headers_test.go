package utils_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/AvengeMedia/dankcalendar/core/utils"
)

func TestGetIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			"cloudflare header wins",
			map[string]string{
				"CF-Connecting-IP": "1.1.1.1",
				"X-Real-Ip":        "2.2.2.2",
				"X-Forwarded-For":  "3.3.3.3",
			},
			"1.1.1.1",
		},
		{
			"real ip beats forwarded for",
			map[string]string{
				"X-Real-Ip":       "2.2.2.2",
				"X-Forwarded-For": "3.3.3.3",
			},
			"2.2.2.2",
		},
		{
			"forwarded for beats remote addr",
			map[string]string{"X-Forwarded-For": "3.3.3.3"},
			"3.3.3.3",
		},
		{
			"falls back to remote addr",
			nil,
			"192.0.2.1:1234",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			assert.Equal(t, tc.want, utils.GetIPAddress(req))
		})
	}
}
