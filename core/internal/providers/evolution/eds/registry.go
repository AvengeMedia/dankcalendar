package eds

import (
	"fmt"
	"sort"

	"github.com/godbus/dbus/v5"
)

// CalendarSource describes a calendar registered in EDS, drawn from the source
// registry without opening the calendar backend.
type CalendarSource struct {
	UID         string
	DisplayName string
	Color       string
	Backend     string
}

// ListCalendarSources returns every enabled calendar source, skipping task and
// memo lists (which live under different key-file sections).
func (c *Client) ListCalendarSources() ([]CalendarSource, error) {
	mgr := c.conn.Object(c.sourcesName, sourceManagerPath)

	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	call := mgr.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0)
	if call.Err != nil {
		return nil, fmt.Errorf("list eds sources: %w", call.Err)
	}
	if err := call.Store(&managed); err != nil {
		return nil, fmt.Errorf("decode eds sources: %w", err)
	}

	var sources []CalendarSource
	for _, ifaces := range managed {
		props, ok := ifaces[ifaceSource]
		if !ok {
			continue
		}
		uid, _ := props["UID"].Value().(string)
		data, _ := props["Data"].Value().(string)
		if uid == "" {
			continue
		}

		kf := parseKeyFile(data)
		if !kf.hasSection("Calendar") {
			continue
		}

		sources = append(sources, CalendarSource{
			UID:         uid,
			DisplayName: kf.get("Data Source", "DisplayName"),
			Color:       kf.get("Calendar", "Color"),
			Backend:     kf.get("Calendar", "BackendName"),
		})
	}

	sort.Slice(sources, func(i, j int) bool { return sources[i].UID < sources[j].UID })
	return sources, nil
}
