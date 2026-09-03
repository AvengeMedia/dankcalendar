package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIcsFilePath(t *testing.T) {
	cases := []struct {
		arg    string
		path   string
		isFile bool
	}{
		{"/home/me/invite.ics", "/home/me/invite.ics", true},
		{"invite.ics", "invite.ics", true},
		{"file:///run/user/1000/doc/ab12/Team%20sync.ics", "/run/user/1000/doc/ab12/Team sync.ics", true},
		{"/tmp/odd:name.ics", "/tmp/odd:name.ics", true},
		{"webcal://example.com/feed.ics", "", false},
		{"https://example.com/feed.ics", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.arg, func(t *testing.T) {
			path, isFile := icsFilePath(tc.arg)
			assert.Equal(t, tc.isFile, isFile)
			assert.Equal(t, tc.path, path)
		})
	}
}

func TestReadICSFile(t *testing.T) {
	dir := t.TempDir()
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:x\r\nBEGIN:VEVENT\r\nUID:a\r\nDTSTART:20260910T120000Z\r\nDTEND:20260910T130000Z\r\nSUMMARY:x\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	good := filepath.Join(dir, "good.ics")
	require.NoError(t, os.WriteFile(good, []byte(ics), 0o600))
	got, err := readICSFile(good)
	require.NoError(t, err)
	assert.Equal(t, ics, got)

	bad := filepath.Join(dir, "bad.ics")
	require.NoError(t, os.WriteFile(bad, []byte("not a calendar"), 0o600))
	_, err = readICSFile(bad)
	assert.ErrorContains(t, err, "bad.ics")

	huge := filepath.Join(dir, "huge.ics")
	require.NoError(t, os.WriteFile(huge, []byte(strings.Repeat("X", 300<<10)), 0o600))
	_, err = readICSFile(huge)
	assert.ErrorContains(t, err, "exceeds")

	_, err = readICSFile(filepath.Join(dir, "missing.ics"))
	assert.Error(t, err)
}
