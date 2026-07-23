// Package filechooser wraps org.freedesktop.portal.FileChooser so Flatpak
// builds can pick host files/directories without --filesystem=home.
package filechooser

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
)

const (
	portalBusName    = "org.freedesktop.portal.Desktop"
	portalObjectPath = "/org/freedesktop/portal/desktop"
	portalChooser    = "org.freedesktop.portal.FileChooser"
	portalRequest    = "org.freedesktop.portal.Request"
)

type Options struct {
	Title       string
	Directory   bool
	AcceptLabel string
	// Glob filters like "*.json". Empty means all files.
	Filters []string
}

// Open shows a portal file chooser and returns local filesystem paths.
// Cancelled dialogs return (nil, nil).
func Open(opts Options) ([]string, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("session bus: %w", err)
	}
	defer conn.Close()

	title := opts.Title
	if title == "" {
		if opts.Directory {
			title = "Select folder"
		} else {
			title = "Select file"
		}
	}

	token := "dankcal" + strings.ReplaceAll(uuid.NewString(), "-", "")
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"modal":        dbus.MakeVariant(true),
		"multiple":     dbus.MakeVariant(false),
		"directory":    dbus.MakeVariant(opts.Directory),
	}
	if opts.AcceptLabel != "" {
		options["accept_label"] = dbus.MakeVariant(opts.AcceptLabel)
	}
	if len(opts.Filters) > 0 && !opts.Directory {
		// D-Bus type a(sa(us)): [(name, [(0=glob|1=mime, pattern), ...]), ...]
		type rule struct {
			Type    uint32
			Pattern string
		}
		type filter struct {
			Name  string
			Rules []rule
		}
		rules := make([]rule, 0, len(opts.Filters))
		for _, f := range opts.Filters {
			rules = append(rules, rule{Type: 0, Pattern: f})
		}
		options["filters"] = dbus.MakeVariant([]filter{{Name: "Files", Rules: rules}})
	}

	obj := conn.Object(portalBusName, portalObjectPath)
	call := obj.Call(portalChooser+".OpenFile", 0, "", title, options)
	if call.Err != nil {
		return nil, fmt.Errorf("OpenFile: %w", call.Err)
	}
	var handle dbus.ObjectPath
	if err := call.Store(&handle); err != nil {
		return nil, fmt.Errorf("OpenFile handle: %w", err)
	}

	uris, err := waitURIs(conn, handle)
	if err != nil {
		return nil, err
	}
	if uris == nil {
		return nil, nil
	}

	paths := make([]string, 0, len(uris))
	for _, u := range uris {
		p, err := uriToPath(u)
		if err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func waitURIs(conn *dbus.Conn, handle dbus.ObjectPath) ([]string, error) {
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(handle),
		dbus.WithMatchInterface(portalRequest),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return nil, fmt.Errorf("filechooser match: %w", err)
	}

	for sig := range signals {
		if sig.Name != portalRequest+".Response" || len(sig.Body) < 2 {
			continue
		}
		code, ok := sig.Body[0].(uint32)
		if !ok {
			return nil, fmt.Errorf("filechooser: bad response code")
		}
		if code == 1 {
			return nil, nil // cancelled
		}
		if code != 0 {
			return nil, fmt.Errorf("filechooser rejected (code %d)", code)
		}
		results, ok := sig.Body[1].(map[string]dbus.Variant)
		if !ok {
			return nil, fmt.Errorf("filechooser: bad results")
		}
		raw, ok := results["uris"]
		if !ok {
			return nil, fmt.Errorf("filechooser: missing uris")
		}
		uris, ok := raw.Value().([]string)
		if !ok {
			return nil, fmt.Errorf("filechooser: uris type")
		}
		return uris, nil
	}
	return nil, fmt.Errorf("filechooser: connection closed")
}

func uriToPath(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "file" {
		return "", fmt.Errorf("unsupported uri scheme %q", parsed.Scheme)
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("empty path from %q", u)
	}
	return path, nil
}

func Available() bool {
	return os.Getenv("FLATPAK_ID") != ""
}
