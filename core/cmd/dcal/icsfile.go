package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/AvengeMedia/dankcalendar/core/internal/icsimport"
)

// icsFilePath reports whether arg names a local file (a plain path or a
// file:// URL) rather than a subscription URL, and returns the path.
func icsFilePath(arg string) (string, bool) {
	u, err := url.Parse(arg)
	switch {
	case err != nil, u.Scheme == "":
		return arg, true
	case u.Scheme == "file":
		if u.Host != "" && u.Host != "localhost" || u.RawQuery != "" || u.Fragment != "" {
			return "", true
		}
		return u.Path, true
	default:
		return "", false
	}
}

func readICSFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s: calendar input must be a regular file", path)
	}
	if info.Size() > icsimport.MaxBytes {
		return "", fmt.Errorf("%s: file exceeds %d KiB", path, icsimport.MaxBytes>>10)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, icsimport.MaxBytes+1))
	if err != nil {
		return "", err
	}
	if _, err := icsimport.Parse(data); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return string(data), nil
}

func openCalendarParams(arg string) (string, map[string]any, error) {
	path, isFile := icsFilePath(arg)
	if !isFile {
		url, err := icsimport.SubscriptionURL(arg)
		return "ui.open", map[string]any{"url": url}, err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil, fmt.Errorf("invalid local calendar file URL")
	}
	data, err := readICSFile(path)
	if err != nil {
		return "", nil, err
	}
	return "ui.openIcs", map[string]any{"ics": data, "name": filepath.Base(path)}, nil
}
