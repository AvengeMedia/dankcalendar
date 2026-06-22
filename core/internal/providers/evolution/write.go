package evolution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	ical "github.com/emersion/go-ical"
	"github.com/google/uuid"

	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/evolution/eds"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/icalconv"
)

var errReadOnly = errors.New("calendar is read-only in evolution data server")

func (p *Provider) CreateEvent(ctx context.Context, cal calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	c, err := p.writable(cal.RemoteID)
	if err != nil {
		return nil, err
	}

	uid := ev.UID
	if uid == "" {
		uid = uuid.NewString()
	}
	ics, err := encodeICS(ev, uid)
	if err != nil {
		return nil, err
	}

	uids, err := c.CreateObjects([]string{ics})
	if err != nil {
		return nil, err
	}
	if len(uids) > 0 && uids[0] != "" {
		uid = uids[0]
	}
	return storedEvent(cal, uid, ev), nil
}

func (p *Provider) UpdateEvent(ctx context.Context, cal calendar.Calendar, ev *calendar.Event) (*calendar.Event, error) {
	if ev.UID == "" {
		return nil, fmt.Errorf("event missing UID")
	}
	c, err := p.writable(cal.RemoteID)
	if err != nil {
		return nil, err
	}

	ics, err := encodeICS(ev, ev.UID)
	if err != nil {
		return nil, err
	}
	if err := c.ModifyObjects([]string{ics}, eds.ModAll); err != nil {
		return nil, err
	}
	return storedEvent(cal, ev.UID, ev), nil
}

func (p *Provider) DeleteEvent(ctx context.Context, cal calendar.Calendar, ev calendar.Event) error {
	if ev.UID == "" {
		return fmt.Errorf("event missing UID")
	}
	c, err := p.writable(cal.RemoteID)
	if err != nil {
		return err
	}
	return c.RemoveObjects([]eds.UIDRID{{UID: ev.UID, RID: ev.RecurringID}}, eds.ModAll)
}

func (p *Provider) writable(uid string) (*eds.Calendar, error) {
	c, err := p.calendar(uid)
	if err != nil {
		return nil, err
	}
	switch writable, err := c.Writable(); {
	case err != nil:
		return nil, err
	case !writable:
		return nil, errReadOnly
	}
	return c, nil
}

// encodeICS returns a bare VEVENT. EDS parses each create/modify argument as a
// single component and rejects a VCALENDAR wrapper, mirroring the bare VEVENTs
// GetObjectList hands back.
func encodeICS(ev *calendar.Event, uid string) (string, error) {
	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(icalconv.CalendarFromEvent(ev, uid)); err != nil {
		return "", fmt.Errorf("encode event: %w", err)
	}

	doc := buf.String()
	begin := strings.Index(doc, "BEGIN:VEVENT")
	end := strings.LastIndex(doc, "END:VEVENT")
	if begin < 0 || end < 0 {
		return "", fmt.Errorf("encode event: no VEVENT produced")
	}
	return doc[begin : end+len("END:VEVENT")], nil
}

func storedEvent(cal calendar.Calendar, uid string, ev *calendar.Event) *calendar.Event {
	out := *ev
	out.CalendarID = cal.ID
	out.UID = uid
	out.RemoteID = uid
	return &out
}
