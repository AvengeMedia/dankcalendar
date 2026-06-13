// Package notify sends desktop notifications over the session bus using the
// org.freedesktop.Notifications interface and dispatches action invocations
// back to the daemon through a single long-lived connection.
package notify

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
)

const (
	busName    = "org.freedesktop.Notifications"
	objectPath = "/org/freedesktop/Notifications"
	appName    = "Dank Calendar"
	appIcon    = "dankcal"
	// desktopEntry must match the installed .desktop file basename.
	desktopEntry = "com.danklinux.dankcal"
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
}

// Client owns the session bus connection. Action and close callbacks are
// invoked from the signal goroutine, one signal at a time.
type Client struct {
	conn     *dbus.Conn
	onAction func(id uint32, action string)
	onClosed func(id uint32)

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
		"urgency":       dbus.MakeVariant(byte(1)),
	}

	// 0 keeps the notification until dismissed; -1 uses the server default.
	timeout := int32(-1)
	if n.Resident {
		timeout = 0
	}

	obj := c.conn.Object(busName, objectPath)
	call := obj.Call(busName+".Notify", 0,
		appName, n.ReplacesID, appIcon, n.Summary, n.Body, actions, hints, timeout)
	if call.Err != nil {
		return 0, fmt.Errorf("send notification: %w", call.Err)
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		return 0, fmt.Errorf("read notification id: %w", err)
	}
	return id, nil
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
