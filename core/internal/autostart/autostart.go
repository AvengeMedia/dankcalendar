package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const entryName = "com.danklinux.dankcalendar.desktop"

func EntryPath() string {
	return filepath.Join(xdg.ConfigHome, "autostart", entryName)
}

func Enabled() bool {
	info, err := os.Stat(EntryPath())
	return err == nil && !info.IsDir()
}

func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	path := EntryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create autostart dir: %w", err)
	}

	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Dank Calendar
Comment=Dank Calendar background service
Exec=%s run -d --hidden
Icon=dankcalendar
Terminal=false
X-GNOME-Autostart-enabled=true
`, exe)
	return os.WriteFile(path, []byte(entry), 0o644)
}

func Disable() error {
	err := os.Remove(EntryPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
