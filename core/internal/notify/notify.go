// Package notify sends desktop notifications over the session bus using the
// org.freedesktop.Notifications interface and dispatches action invocations
// back to the daemon through a single long-lived connection.
package notify

import (
	"fmt"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/log"
)

const (
	busName    = "org.freedesktop.Notifications"
	objectPath = "/org/freedesktop/Notifications"
	appName    = "Dank Calendar"
	appIcon    = "dankcalendar"
	// desktopEntry must match the installed .desktop file basename.
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

func notificationIcon() string {
	return iconForEnv(os.Getenv("FLATPAK_ID"))
}

// Flatpak only exports icons named after the app ID.
func iconForEnv(flatpakID string) string {
	if flatpakID != "" {
		return desktopEntry
	}
	return appIcon
}

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
	Summary    string
	Body       string
	Actions    []Action
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

// Client owns the session bus connection. Action and close callbacks are
// invoked from the signal goroutine, one signal at a time.
type Client struct {
	conn     *dbus.Conn
	onAction func(id uint32, action string)
	onClosed func(id uint32)
	sound    soundPlayer

	mu     sync.Mutex
	closed bool
}

func New() (*Client, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect session bus: %w", err)
	}

	matchOpts := []dbus.MatchOption{
		dbus.WithMatchInterface(busName),
		dbus.WithMatchObjectPath(objectPath),
	}
	if err := conn.AddMatchSignal(matchOpts...); err != nil {
		conn.Close()
		return nil, fmt.Errorf("subscribe notification signals: %w", err)
	}

	c := &Client{conn: conn}

	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	go c.dispatch(signals)

	return c, nil
}

// SetHandlers must be called before the first Send.
func (c *Client) SetHandlers(onAction func(id uint32, action string), onClosed func(id uint32)) {
	c.onAction = onAction
	c.onClosed = onClosed
}

func (c *Client) Send(n Notification) (uint32, error) {
	actions := make([]string, 0, len(n.Actions)*2)
	for _, a := range n.Actions {
		actions = append(actions, a.Key, a.Label)
	}

	hints := map[string]dbus.Variant{
		"desktop-entry": dbus.MakeVariant(desktopEntry),
		"urgency":       dbus.MakeVariant(n.Urgency),
	}
	if n.SoundName != "" {
		if c.sound.play(n.SoundName) {
			hints["suppress-sound"] = dbus.MakeVariant(true)
		} else {
			hints["sound-name"] = dbus.MakeVariant(n.SoundName)
		}
	}

	// 0 keeps the notification until dismissed; -1 uses the server default.
	timeout := int32(-1)
	if n.Resident {
		timeout = 0
	}

	obj := c.conn.Object(busName, objectPath)
	call := obj.Call(busName+".Notify", 0,
		appName, n.ReplacesID, notificationIcon(), n.Summary, n.Body, actions, hints, timeout)
	if call.Err != nil {
		return 0, fmt.Errorf("send notification: %w", call.Err)
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		return 0, fmt.Errorf("read notification id: %w", err)
	}
	return id, nil
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

func (c *Client) Dismiss(id uint32) {
	obj := c.conn.Object(busName, objectPath)
	if call := obj.Call(busName+".CloseNotification", 0, id); call.Err != nil {
		log.Debugf("close notification %d: %v", id, call.Err)
	}
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

func (c *Client) dispatch(signals <-chan *dbus.Signal) {
	for sig := range signals {
		switch sig.Name {
		case busName + ".ActionInvoked":
			if len(sig.Body) < 2 || c.onAction == nil {
				continue
			}
			id, idOK := sig.Body[0].(uint32)
			action, actionOK := sig.Body[1].(string)
			if !idOK || !actionOK {
				continue
			}
			c.onAction(id, action)
		case busName + ".NotificationClosed":
			if len(sig.Body) < 1 || c.onClosed == nil {
				continue
			}
			id, ok := sig.Body[0].(uint32)
			if !ok {
				continue
			}
			c.onClosed(id)
		}
	}
}
