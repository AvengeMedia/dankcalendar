package reminders

import (
	"context"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/log"
)

const (
	login1Path      = "/org/freedesktop/login1"
	login1Manager   = "org.freedesktop.login1.Manager"
	prepareForSleep = login1Manager + ".PrepareForSleep"
)

// watchSuspend wakes the engine on resume from sleep. Timers run on the
// monotonic clock, which pauses while suspended, so a parked wakeup would
// otherwise fire late by the suspended duration.
func (e *Engine) watchSuspend(ctx context.Context) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Warnf("reminders: suspend watch unavailable: %v", err)
		return
	}
	defer conn.Close()

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(login1Path),
		dbus.WithMatchInterface(login1Manager),
		dbus.WithMatchMember("PrepareForSleep"),
	); err != nil {
		log.Warnf("reminders: suspend match: %v", err)
		return
	}

	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stop:
			return
		case sig, ok := <-signals:
			if !ok || sig.Name != prepareForSleep || len(sig.Body) == 0 {
				continue
			}
			// Arg is true entering sleep, false on resume.
			if preparing, _ := sig.Body[0].(bool); !preparing {
				e.Wake()
			}
		}
	}
}
