// Package eds is a pure-Go client for Evolution Data Server's D-Bus API. It
// talks to the source registry and the calendar factory directly over the
// session bus, so the calendar provider needs no cgo bindings to libecal.
package eds

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	sourcesPrefix  = "org.gnome.evolution.dataserver.Sources"
	calendarPrefix = "org.gnome.evolution.dataserver.Calendar"

	sourceManagerPath   = dbus.ObjectPath("/org/gnome/evolution/dataserver/SourceManager")
	calendarFactoryPath = dbus.ObjectPath("/org/gnome/evolution/dataserver/CalendarFactory")

	ifaceSource   = "org.gnome.evolution.dataserver.Source"
	ifaceFactory  = "org.gnome.evolution.dataserver.CalendarFactory"
	ifaceCalendar = "org.gnome.evolution.dataserver.Calendar"
)

// ErrUnavailable means Evolution Data Server is not reachable on the session
// bus, either because it is not installed or there is no session bus at all.
var ErrUnavailable = errors.New("evolution data server is not available on the session bus")

// Client owns a session-bus connection and the resolved, version-suffixed
// service names (e.g. Sources5, Calendar8).
type Client struct {
	conn         *dbus.Conn
	sourcesName  string
	calendarName string
}

func Dial() (*Client, error) {
	// A private connection: the shared dbus.SessionBus() is held by other
	// components (notifications), and closing it here would tear theirs down.
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	names, err := busNames(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	sources := highestVersioned(names, sourcesPrefix)
	calendar := highestVersioned(names, calendarPrefix)
	if sources == "" || calendar == "" {
		conn.Close()
		return nil, ErrUnavailable
	}

	return &Client{conn: conn, sourcesName: sources, calendarName: calendar}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Available reports whether Evolution Data Server is reachable on the session
// bus, so callers can hide the provider where it makes no sense.
func Available() bool {
	c, err := Dial()
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// busNames merges the activatable and currently-owned names so a not-yet-started
// EDS is still discovered (the factory is D-Bus activated on first use).
func busNames(conn *dbus.Conn) ([]string, error) {
	bus := conn.BusObject()

	var activatable []string
	if err := bus.Call("org.freedesktop.DBus.ListActivatableNames", 0).Store(&activatable); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	var owned []string
	if err := bus.Call("org.freedesktop.DBus.ListNames", 0).Store(&owned); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return append(activatable, owned...), nil
}

// highestVersioned returns the prefix match with the largest numeric suffix so
// the newest available EDS API generation wins across releases.
func highestVersioned(names []string, prefix string) string {
	best := ""
	bestVer := -1
	for _, n := range names {
		if !strings.HasPrefix(n, prefix) {
			continue
		}
		ver, err := strconv.Atoi(strings.TrimPrefix(n, prefix))
		if err != nil {
			continue
		}
		if ver > bestVer {
			best, bestVer = n, ver
		}
	}
	return best
}
