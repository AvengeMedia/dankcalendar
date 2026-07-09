// Package uriopen opens a URI with its default XDG handler.
// Qt.openUrlExternally doesn't work with geo: URIs for some reason, so we
// made our own opener.
package uriopen

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
)

type Opener struct{}

func (Opener) OpenURI(uri string) error { return Open(uri) }

func Open(uri string) error {
	scheme, _, ok := strings.Cut(uri, ":")
	if !ok || scheme == "" {
		return fmt.Errorf("invalid uri %q", uri)
	}

	appDirs := applicationDirs()
	id := defaultHandler("x-scheme-handler/"+strings.ToLower(scheme), mimeappsPaths(), appDirs)
	if id == "" {
		return fmt.Errorf("no handler registered for scheme %q", scheme)
	}
	entry := findDesktopFile(id, appDirs)
	if entry == "" {
		return fmt.Errorf("desktop entry %q not found", id)
	}
	execLine, err := desktopExec(entry)
	if err != nil {
		return fmt.Errorf("%s: %w", entry, err)
	}
	argv, err := buildArgv(execLine, uri)
	if err != nil {
		return fmt.Errorf("%s: %w", entry, err)
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func mimeappsPaths() []string {
	paths := []string{filepath.Join(xdg.ConfigHome, "mimeapps.list")}
	for _, dir := range xdg.ConfigDirs {
		paths = append(paths, filepath.Join(dir, "mimeapps.list"))
	}
	paths = append(paths, filepath.Join(xdg.DataHome, "applications", "mimeapps.list"))
	for _, dir := range xdg.DataDirs {
		paths = append(paths, filepath.Join(dir, "applications", "mimeapps.list"))
	}
	return paths
}

func applicationDirs() []string {
	dirs := []string{filepath.Join(xdg.DataHome, "applications")}
	for _, dir := range xdg.DataDirs {
		dirs = append(dirs, filepath.Join(dir, "applications"))
	}
	return dirs
}

// defaultHandler resolves a mime type to a desktop-file id via mimeapps.list,
// falling back to mimeinfo.cache; entries with no desktop file are skipped.
func defaultHandler(mime string, mimeapps []string, appDirs []string) string {
	for _, path := range mimeapps {
		for _, id := range iniValues(path, "Default Applications", mime) {
			if findDesktopFile(id, appDirs) != "" {
				return id
			}
		}
	}
	for _, dir := range appDirs {
		for _, id := range iniValues(filepath.Join(dir, "mimeinfo.cache"), "MIME Cache", mime) {
			if findDesktopFile(id, appDirs) != "" {
				return id
			}
		}
	}
	return ""
}

// iniValues returns the ;-separated values of key within [section].
func iniValues(path, section, key string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			inSection = line == "["+section+"]"
		case inSection:
			k, v, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(k) != key {
				continue
			}
			var out []string
			for _, id := range strings.Split(v, ";") {
				if id = strings.TrimSpace(id); id != "" {
					out = append(out, id)
				}
			}
			return out
		}
	}
	return nil
}

// findDesktopFile locates a desktop-file id; per spec a dash may stand for a
// subdirectory (vendor-foo.desktop == vendor/foo.desktop).
func findDesktopFile(id string, appDirs []string) string {
	for _, dir := range appDirs {
		if path := filepath.Join(dir, id); fileExists(path) {
			return path
		}
		for i := 0; i < len(id); i++ {
			if id[i] != '-' {
				continue
			}
			if path := filepath.Join(dir, id[:i], id[i+1:]); fileExists(path) {
				return path
			}
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func desktopExec(path string) (string, error) {
	raw := iniValue(path, "Desktop Entry", "Exec")
	if raw == "" {
		return "", errors.New("no Exec key")
	}
	return raw, nil
}

// iniValue returns a single raw value; Exec lines legitimately contain
// semicolons, so no splitting.
func iniValue(path, section, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			inSection = line == "["+section+"]"
		case inSection:
			k, v, ok := strings.Cut(line, "=")
			if ok && strings.TrimSpace(k) == key {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// buildArgv tokenizes an Exec value per the Desktop Entry spec and expands
// field codes; without a field code the URI is appended.
func buildArgv(execLine, uri string) ([]string, error) {
	tokens, err := splitExec(execLine)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, errors.New("empty Exec")
	}

	argv := make([]string, 0, len(tokens)+1)
	placed := false
	for _, tok := range tokens {
		switch tok {
		case "%u", "%U", "%f", "%F":
			argv = append(argv, uri)
			placed = true
			continue
		case "%i", "%c", "%k", "%d", "%D", "%n", "%N", "%v", "%m":
			continue
		}
		expanded, used := expandFieldCodes(tok, uri)
		placed = placed || used
		argv = append(argv, expanded)
	}
	if !placed {
		argv = append(argv, uri)
	}
	return argv, nil
}

func expandFieldCodes(tok, uri string) (string, bool) {
	var out strings.Builder
	used := false
	for i := 0; i < len(tok); i++ {
		if tok[i] != '%' || i+1 >= len(tok) {
			out.WriteByte(tok[i])
			continue
		}
		i++
		switch tok[i] {
		case 'u', 'U', 'f', 'F':
			out.WriteString(uri)
			used = true
		case '%':
			out.WriteByte('%')
		default:
		}
	}
	return out.String(), used
}

func splitExec(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inQuote, escaped, started := false, false, false

	flush := func() {
		if started || cur.Len() > 0 {
			args = append(args, cur.String())
			cur.Reset()
			started = false
		}
	}

	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case inQuote && r == '\\':
			escaped = true
		case r == '"':
			inQuote = !inQuote
			started = true
		case !inQuote && (r == ' ' || r == '\t'):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if inQuote || escaped {
		return nil, errors.New("unterminated quoting in Exec")
	}
	flush()
	return args, nil
}
