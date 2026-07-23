package notify

import (
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/godbus/dbus/v5"

	"github.com/AvengeMedia/dankgo/log"
	"github.com/AvengeMedia/dankgo/portal"
)

const portalIDPrefix = "dankcal-"

type portalBackend struct {
	client   *portal.NotificationClient
	onAction func(id uint32, action string)
	onClosed func(id uint32)
	sound    soundPlayer
	nextID   atomic.Uint32
}

func newPortalBackend(conn *dbus.Conn) (*portalBackend, error) {
	client, err := portal.NewNotificationClient(conn)
	if err != nil {
		return nil, err
	}
	return &portalBackend{client: client}, nil
}

func (b *portalBackend) setHandlers(onAction func(id uint32, action string), onClosed func(id uint32)) {
	b.onAction = onAction
	b.onClosed = onClosed
	b.client.SetActionHandler(b.handleAction)
}

func (b *portalBackend) handleAction(portalID, action string) {
	id, ok := parsePortalID(portalID)
	if !ok {
		return
	}
	if b.onAction != nil {
		b.onAction(id, action)
	}
	// The portal withdraws a notification on activation and has no
	// NotificationClosed signal, so closure is synthesized to keep the
	// engines' pending state in sync.
	if b.onClosed != nil {
		b.onClosed(id)
	}
}

func (b *portalBackend) send(n Notification) (uint32, error) {
	id := n.ReplacesID
	if id == 0 {
		id = b.nextID.Add(1)
	}

	pn := portal.Notification{
		Title:    n.Summary,
		Body:     n.Body,
		Priority: priorityFor(n.Urgency),
	}
	for _, a := range n.Actions {
		if a.Key == "default" {
			pn.DefaultAction = a.Key
			continue
		}
		pn.Buttons = append(pn.Buttons, portal.Button{Label: a.Label, Action: a.Key})
	}
	if n.SoundName != "" {
		pn.Sound = portal.SoundDefault
		if b.sound.play(n.SoundName) {
			pn.Sound = portal.SoundSilence
		}
	}

	if err := b.client.Add(formatPortalID(id), pn); err != nil {
		return 0, err
	}
	return id, nil
}

func (b *portalBackend) dismiss(id uint32) {
	if err := b.client.Remove(formatPortalID(id)); err != nil {
		log.Debugf("remove portal notification %d: %v", id, err)
	}
	if b.onClosed != nil {
		b.onClosed(id)
	}
}

func formatPortalID(id uint32) string {
	return portalIDPrefix + strconv.FormatUint(uint64(id), 10)
}

func parsePortalID(s string) (uint32, bool) {
	raw, ok := strings.CutPrefix(s, portalIDPrefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(id), true
}

func priorityFor(urgency byte) string {
	switch urgency {
	case UrgencyLow:
		return portal.PriorityLow
	case UrgencyCritical:
		return portal.PriorityUrgent
	default:
		return portal.PriorityNormal
	}
}
