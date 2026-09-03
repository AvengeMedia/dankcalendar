package main

import (
	"fmt"
	"net/url"
	"os"

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
	if info.Size() > icsimport.MaxBytes {
		return "", fmt.Errorf("%s: file exceeds %d KiB", path, icsimport.MaxBytes>>10)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if _, err := icsimport.Parse(data); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return string(data), nil
}
