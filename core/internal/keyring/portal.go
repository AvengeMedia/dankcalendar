package keyring

import (
	"fmt"
	"io"
	"os"

	"github.com/godbus/dbus/v5"
)

const (
	portalBusName    = "org.freedesktop.portal.Desktop"
	portalObjectPath = "/org/freedesktop/portal/desktop"
	portalSecret     = "org.freedesktop.portal.Secret"
	portalRequest    = "org.freedesktop.portal.Request"
)

// RetrievePortalSecret asks the XDG Secret portal for the per-app master secret.
// Used under Flatpak so credentials can be encrypted without talking to
// org.freedesktop.secrets directly.
func RetrievePortalSecret() ([]byte, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("session bus: %w", err)
	}
	defer conn.Close()

	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("secret pipe: %w", err)
	}
	defer r.Close()

	obj := conn.Object(portalBusName, portalObjectPath)
	call := obj.Call(portalSecret+".RetrieveSecret", 0, dbus.UnixFD(w.Fd()), map[string]dbus.Variant{})
	if call.Err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("RetrieveSecret: %w", call.Err)
	}

	var handle dbus.ObjectPath
	if err := call.Store(&handle); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("RetrieveSecret handle: %w", err)
	}

	if err := waitRequestSuccess(conn, handle); err != nil {
		_ = w.Close()
		return nil, err
	}
	_ = w.Close()

	secret, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read portal secret: %w", err)
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("portal returned empty secret")
	}
	return secret, nil
}

func waitRequestSuccess(conn *dbus.Conn, handle dbus.ObjectPath) error {
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(handle),
		dbus.WithMatchInterface(portalRequest),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return fmt.Errorf("secret request match: %w", err)
	}

	for sig := range signals {
		if sig.Name != portalRequest+".Response" || len(sig.Body) < 1 {
			continue
		}
		code, ok := sig.Body[0].(uint32)
		if !ok {
			return fmt.Errorf("secret request: bad response code")
		}
		if code != 0 {
			return fmt.Errorf("secret request rejected (code %d)", code)
		}
		return nil
	}
	return fmt.Errorf("secret request: connection closed")
}

func InFlatpak() bool {
	return os.Getenv("FLATPAK_ID") != ""
}
