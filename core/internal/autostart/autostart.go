package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/portal"
)

const (
	entryName = "com.danklinux.dankcalendar.desktop"

	portalBusName    = "org.freedesktop.portal.Desktop"
	portalObjectPath = "/org/freedesktop/portal/desktop"
	portalBackground = "org.freedesktop.portal.Background.RequestBackground"
	portalTimeout    = 2 * time.Minute
)

var daemonArgs = []string{"run", "-d", "--hidden"}

func EntryPath() string {
	return filepath.Join(xdg.ConfigHome, "autostart", entryName)
}

// The Background portal has no state query, so a sandbox-local marker records
// whether autostart was granted.
func flatpakMarkerPath() string {
	return filepath.Join(xdg.ConfigHome, "dankcalendar", "autostart-granted")
}

func desktopEntry(exec string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Dank Calendar
Comment=Dank Calendar background service
Exec=%s
Icon=com.danklinux.dankcalendar
Terminal=false
X-GNOME-Autostart-enabled=true
`, exec)
}

func Enabled() bool {
	if portal.InFlatpak() {
		info, err := os.Stat(flatpakMarkerPath())
		return err == nil && !info.IsDir()
	}
	info, err := os.Stat(EntryPath())
	return err == nil && !info.IsDir()
}

func Enable() error {
	if portal.InFlatpak() {
		return enableFlatpak()
	}
	return enableHost()
}

func Disable() error {
	if portal.InFlatpak() {
		return disableFlatpak()
	}
	return disableHost()
}

func enableHost() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	path := EntryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create autostart dir: %w", err)
	}

	exec := exe + " " + strings.Join(daemonArgs, " ")
	return os.WriteFile(path, []byte(desktopEntry(exec)), 0o644)
}

func disableHost() error {
	err := os.Remove(EntryPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func enableFlatpak() error {
	if err := requestBackground(true); err != nil {
		return err
	}
	path := flatpakMarkerPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create autostart state dir: %w", err)
	}
	return os.WriteFile(path, nil, 0o644)
}

func disableFlatpak() error {
	reqErr := requestBackground(false)
	if err := os.Remove(flatpakMarkerPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return reqErr
}

// requestBackground runs the org.freedesktop.portal.Background handshake on a
// private connection and blocks until the async Response signal arrives.
func requestBackground(autostart bool) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect session bus: %w", err)
	}
	defer conn.Close()

	token := fmt.Sprintf("dcal_autostart_%d", time.Now().UnixNano())
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	handle := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)

	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
		dbus.WithMatchObjectPath(handle),
	); err != nil {
		return fmt.Errorf("subscribe portal response: %w", err)
	}

	responses := make(chan *dbus.Signal, 1)
	conn.Signal(responses)

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"reason":       dbus.MakeVariant("Start Dank Calendar in the background at login."),
		"autostart":    dbus.MakeVariant(autostart),
		"commandline":  dbus.MakeVariant(append([]string{"dcal"}, daemonArgs...)),
	}

	obj := conn.Object(portalBusName, portalObjectPath)
	var reqPath dbus.ObjectPath
	if call := obj.Call(portalBackground, 0, "", options); call.Err != nil {
		return fmt.Errorf("request background: %w", call.Err)
	} else if err := call.Store(&reqPath); err != nil {
		return fmt.Errorf("read background request handle: %w", err)
	}

	timeout := time.NewTimer(portalTimeout)
	defer timeout.Stop()
	for {
		select {
		case sig := <-responses:
			if sig.Path != handle || len(sig.Body) < 1 {
				continue
			}
			code, ok := sig.Body[0].(uint32)
			if !ok {
				return fmt.Errorf("malformed background portal response")
			}
			if code != 0 {
				return fmt.Errorf("background portal request was not granted (code %d)", code)
			}
			return nil
		case <-timeout.C:
			return fmt.Errorf("timed out waiting for background portal response")
		}
	}
}
