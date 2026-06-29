package eds

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// ModAll targets every instance of an event, the mode used for whole-event
// modify and remove operations.
const ModAll = "all"

// Calendar is an opened EDS calendar backend. EDS spawns a per-calendar
// subprocess that stays alive only while the owning connection is open, so the
// handle keeps a reference to the Client's connection.
type Calendar struct {
	obj dbus.BusObject
}

// UIDRID identifies an event, optionally a single recurrence instance by its
// recurrence-id. An empty RID targets the master event.
type UIDRID struct {
	UID string
	RID string
}

// OpenCalendar resolves the event-calendar backend through the factory and opens
// it.
func (c *Client) OpenCalendar(uid string) (*Calendar, error) {
	return c.openBackend("OpenCalendar", uid)
}

// OpenTaskList resolves the task-list backend. EDS exposes task lists over the
// same Calendar interface, so the handle reads/writes VTODO just like VEVENT.
func (c *Client) OpenTaskList(uid string) (*Calendar, error) {
	return c.openBackend("OpenTaskList", uid)
}

func (c *Client) openBackend(method, uid string) (*Calendar, error) {
	factory := c.conn.Object(c.calendarName, calendarFactoryPath)

	var objPath, busName string
	call := factory.Call(ifaceFactory+"."+method, 0, uid)
	if call.Err != nil {
		return nil, fmt.Errorf("eds %s %q: %w", method, uid, call.Err)
	}
	if err := call.Store(&objPath, &busName); err != nil {
		return nil, fmt.Errorf("eds %s %q: %w", method, uid, err)
	}

	cal := &Calendar{obj: c.conn.Object(busName, dbus.ObjectPath(objPath))}
	var props []string
	if call := cal.obj.Call(ifaceCalendar+".Open", 0); call.Err != nil {
		return nil, fmt.Errorf("eds %s %q: %w", method, uid, call.Err)
	} else if err := call.Store(&props); err != nil {
		return nil, fmt.Errorf("eds %s %q: %w", method, uid, err)
	}
	return cal, nil
}

func (c *Calendar) Refresh() error {
	if call := c.obj.Call(ifaceCalendar+".Refresh", 0); call.Err != nil {
		return fmt.Errorf("refresh eds calendar: %w", call.Err)
	}
	return nil
}

func (c *Calendar) Revision() (string, error) {
	return c.stringProp("Revision")
}

func (c *Calendar) Writable() (bool, error) {
	v, err := c.obj.GetProperty(ifaceCalendar + ".Writable")
	if err != nil {
		return false, err
	}
	writable, _ := v.Value().(bool)
	return writable, nil
}

// ObjectList returns the iCalendar objects matching an S-expression query, e.g.
// "#t" for every object. Each entry is a bare VEVENT component.
func (c *Calendar) ObjectList(query string) ([]string, error) {
	var objects []string
	call := c.obj.Call(ifaceCalendar+".GetObjectList", 0, query)
	if call.Err != nil {
		return nil, fmt.Errorf("query eds calendar: %w", call.Err)
	}
	if err := call.Store(&objects); err != nil {
		return nil, fmt.Errorf("decode eds objects: %w", err)
	}
	return objects, nil
}

// CreateObjects adds new iCalendar objects and returns their assigned UIDs.
func (c *Calendar) CreateObjects(ics []string) ([]string, error) {
	var uids []string
	call := c.obj.Call(ifaceCalendar+".CreateObjects", 0, ics, uint32(0))
	if call.Err != nil {
		return nil, fmt.Errorf("create eds objects: %w", call.Err)
	}
	if err := call.Store(&uids); err != nil {
		return nil, fmt.Errorf("decode created uids: %w", err)
	}
	return uids, nil
}

func (c *Calendar) ModifyObjects(ics []string, mod string) error {
	if call := c.obj.Call(ifaceCalendar+".ModifyObjects", 0, ics, mod, uint32(0)); call.Err != nil {
		return fmt.Errorf("modify eds objects: %w", call.Err)
	}
	return nil
}

func (c *Calendar) RemoveObjects(ids []UIDRID, mod string) error {
	if call := c.obj.Call(ifaceCalendar+".RemoveObjects", 0, ids, mod, uint32(0)); call.Err != nil {
		return fmt.Errorf("remove eds objects: %w", call.Err)
	}
	return nil
}

func (c *Calendar) Close() {
	c.obj.Call(ifaceCalendar+".Close", 0)
}

func (c *Calendar) stringProp(name string) (string, error) {
	v, err := c.obj.GetProperty(ifaceCalendar + "." + name)
	if err != nil {
		return "", err
	}
	s, _ := v.Value().(string)
	return s, nil
}
