package notify

import (
	"fmt"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/log"
)

const (
	fdoBusName    = "org.freedesktop.Notifications"
	fdoObjectPath = "/org/freedesktop/Notifications"
)

type fdoBackend struct {
	conn     *dbus.Conn
	onAction func(id uint32, action string)
	onClosed func(id uint32)
	sound    soundPlayer
}

func newFDOBackend(conn *dbus.Conn) (*fdoBackend, error) {
	matchOpts := []dbus.MatchOption{
		dbus.WithMatchInterface(fdoBusName),
		dbus.WithMatchObjectPath(fdoObjectPath),
	}
	if err := conn.AddMatchSignal(matchOpts...); err != nil {
		return nil, fmt.Errorf("subscribe notification signals: %w", err)
	}

	b := &fdoBackend{conn: conn}
	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	go b.dispatch(signals)
	return b, nil
}

func (b *fdoBackend) setHandlers(onAction func(id uint32, action string), onClosed func(id uint32)) {
	b.onAction = onAction
	b.onClosed = onClosed
}

func (b *fdoBackend) send(n Notification) (uint32, error) {
	actions := make([]string, 0, len(n.Actions)*2)
	for _, a := range n.Actions {
		actions = append(actions, a.Key, a.Label)
	}

	hints := map[string]dbus.Variant{
		"desktop-entry": dbus.MakeVariant(desktopEntry),
		"urgency":       dbus.MakeVariant(n.Urgency),
	}
	if n.SoundName != "" {
		if b.sound.play(n.SoundName) {
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

	obj := b.conn.Object(fdoBusName, fdoObjectPath)
	call := obj.Call(fdoBusName+".Notify", 0,
		appName, n.ReplacesID, desktopEntry, n.Summary, n.Body, actions, hints, timeout)
	if call.Err != nil {
		return 0, fmt.Errorf("send notification: %w", call.Err)
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		return 0, fmt.Errorf("read notification id: %w", err)
	}
	return id, nil
}

func (b *fdoBackend) dismiss(id uint32) {
	obj := b.conn.Object(fdoBusName, fdoObjectPath)
	if call := obj.Call(fdoBusName+".CloseNotification", 0, id); call.Err != nil {
		log.Debugf("close notification %d: %v", id, call.Err)
	}
}

func (b *fdoBackend) dispatch(signals <-chan *dbus.Signal) {
	for sig := range signals {
		switch sig.Name {
		case fdoBusName + ".ActionInvoked":
			if len(sig.Body) < 2 || b.onAction == nil {
				continue
			}
			id, idOK := sig.Body[0].(uint32)
			action, actionOK := sig.Body[1].(string)
			if !idOK || !actionOK {
				continue
			}
			b.onAction(id, action)
		case fdoBusName + ".NotificationClosed":
			if len(sig.Body) < 1 || b.onClosed == nil {
				continue
			}
			id, ok := sig.Body[0].(uint32)
			if !ok {
				continue
			}
			b.onClosed(id)
		}
	}
}
