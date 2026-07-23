// Package notify sends desktop notifications and dispatches action
// invocations back to the daemon through a single long-lived session bus
// connection. Sandboxed (Flatpak) builds go through the XDG notification
// portal; native installs talk to org.freedesktop.Notifications directly.
package notify

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/portal"
)

const (
	appName = "Dank Calendar"
	// desktopEntry must match the installed .desktop file and icon basename.
	desktopEntry = "com.danklinux.dankcalendar"

	portalBusName    = "org.freedesktop.portal.Desktop"
	portalObjectPath = "/org/freedesktop/portal/desktop"
	portalOpenURI    = "org.freedesktop.portal.OpenURI.OpenURI"

	// SoundReminder is a freedesktop sound-naming-spec event name; servers
	// that support the sound-name hint resolve it against the active theme.
	SoundReminder = "message-new-instant"
	// SoundTask is intentionally alarm-like so a due task is more noticeable
	// than an ordinary calendar reminder.
	SoundTask = "alarm-clock-elapsed"
)

const (
	UrgencyLow byte = iota
	UrgencyNormal
	UrgencyCritical
)

type Action struct {
	Key   string
	Label string
}

type Notification struct {
	Summary string
	Body    string
	Actions []Action
	// Resident keeps the notification until dismissed; the portal backend
	// ignores it since persistence there is desktop policy.
	Resident   bool
	ReplacesID uint32
	// SoundName is a freedesktop sound-theme event name played on send;
	// empty sends no sound hints so the server's own config applies.
	SoundName string
	// Urgency follows the freedesktop notification specification. A zero value
	// is low urgency, so callers should explicitly use UrgencyNormal when the
	// notification is not priority-sensitive.
	Urgency byte
}

type backend interface {
	setHandlers(onAction func(id uint32, action string), onClosed func(id uint32))
	send(n Notification) (uint32, error)
	dismiss(id uint32)
}

// Client owns the session bus connection. Action and close callbacks are
// invoked from the signal goroutine, one signal at a time.
type Client struct {
	conn *dbus.Conn
	b    backend

	mu     sync.Mutex
	closed bool
}

func New() (*Client, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect session bus: %w", err)
	}

	var b backend
	if portal.InFlatpak() {
		b, err = newPortalBackend(conn)
	} else {
		b, err = newFDOBackend(conn)
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &Client{conn: conn, b: b}, nil
}

// SetHandlers must be called before the first Send.
func (c *Client) SetHandlers(onAction func(id uint32, action string), onClosed func(id uint32)) {
	c.b.setHandlers(onAction, onClosed)
}

func (c *Client) Send(n Notification) (uint32, error) {
	return c.b.send(n)
}

func (c *Client) Dismiss(id uint32) {
	c.b.dismiss(id)
}

// OpenURI hands a URI to the XDG desktop portal so the user's configured handler
// opens it, reusing the session bus instead of spawning a helper process. The
// empty parent-window token is valid; options are intentionally unset.
func (c *Client) OpenURI(uri string) error {
	if uri == "" {
		return nil
	}
	obj := c.conn.Object(portalBusName, portalObjectPath)
	call := obj.Call(portalOpenURI, 0, "", uri, map[string]dbus.Variant{})
	if call.Err != nil {
		return fmt.Errorf("open uri via portal: %w", call.Err)
	}
	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.conn.Close()
}
