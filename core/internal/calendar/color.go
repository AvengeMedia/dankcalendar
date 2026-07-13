package calendar

import "strings"

// NormalizeColor reduces the color forms providers report to what the GUI
// consumes: hex with or without the leading '#' (Apple's #rrggbbaa drops the
// alpha byte, X11's 16-bit #rrrrggggbbbb keeps the high byte per channel), and
// CSS color names as RFC 7986 COLOR publishes them. Anything else is dropped
// so the GUI falls back to its palette.
func NormalizeColor(raw string) string {
	s := strings.TrimSpace(raw)
	hex := strings.TrimPrefix(s, "#")
	if hex == "" {
		return ""
	}
	if !isHex(hex) {
		if s == hex && isAlpha(s) {
			return strings.ToLower(s)
		}
		return ""
	}

	switch len(hex) {
	case 3, 6:
	case 8:
		hex = hex[:6]
	case 12:
		hex = hex[0:2] + hex[4:6] + hex[8:10]
	default:
		return ""
	}
	return "#" + strings.ToLower(hex)
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func isAlpha(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		default:
			return false
		}
	}
	return true
}
