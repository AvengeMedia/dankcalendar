package colorscheme

import (
	"sync"

	"github.com/AvengeMedia/dankgo/log"
	"github.com/godbus/dbus/v5"
)

const (
	Topic = "colorScheme"

	portalDest     = "org.freedesktop.portal.Desktop"
	portalPath     = "/org/freedesktop/portal/desktop"
	settingsIface  = "org.freedesktop.portal.Settings"
	appearanceNS   = "org.freedesktop.appearance"
	colorSchemeKey = "color-scheme"
)

type Publisher func(topic string, data any)

type State struct {
	Available   bool   `json:"available"`
	ColorScheme uint32 `json:"colorScheme"`
}

// Watcher tracks the freedesktop appearance color-scheme via the desktop
// portal and publishes changes so clients never have to poll or shell out.
type Watcher struct {
	publish Publisher

	mu        sync.RWMutex
	available bool
	scheme    uint32
}

func New(publish Publisher) *Watcher {
	return &Watcher{publish: publish}
}

func (w *Watcher) State() State {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return State{Available: w.available, ColorScheme: w.scheme}
}

func (w *Watcher) Start() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}

	obj := conn.Object(portalDest, portalPath)
	var variant dbus.Variant
	if err := obj.Call(settingsIface+".ReadOne", 0, appearanceNS, colorSchemeKey).Store(&variant); err != nil {
		conn.Close()
		return err
	}
	if scheme, ok := asUint32(variant); ok {
		w.mu.Lock()
		w.available = true
		w.scheme = scheme
		w.mu.Unlock()
	}

	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface(settingsIface),
		dbus.WithMatchMember("SettingChanged"),
	); err != nil {
		conn.Close()
		return err
	}

	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	go w.loop(signals)
	return nil
}

func (w *Watcher) loop(signals chan *dbus.Signal) {
	for sig := range signals {
		if sig.Name != settingsIface+".SettingChanged" || len(sig.Body) < 3 {
			continue
		}

		namespace, _ := sig.Body[0].(string)
		key, _ := sig.Body[1].(string)
		if namespace != appearanceNS || key != colorSchemeKey {
			continue
		}

		variant, ok := sig.Body[2].(dbus.Variant)
		if !ok {
			continue
		}
		scheme, ok := asUint32(variant)
		if !ok {
			continue
		}

		w.mu.Lock()
		changed := !w.available || w.scheme != scheme
		w.available = true
		w.scheme = scheme
		w.mu.Unlock()

		if changed && w.publish != nil {
			w.publish(Topic, w.State())
		}
	}
	log.Debug("color-scheme signal channel closed")
}

func asUint32(v dbus.Variant) (uint32, bool) {
	switch val := v.Value().(type) {
	case uint32:
		return val, true
	case dbus.Variant:
		return asUint32(val)
	}
	return 0, false
}
