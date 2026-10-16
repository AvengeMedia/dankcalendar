package ipc

import "sync"

// PendingOpen queues "ui" actions captured before the GUI subscribed to the
// "ui" topic (e.g. a webcal:// link or an event open that cold-started the
// app). It is flushed onto the bus the moment a "ui" subscriber appears.
type PendingOpen struct {
	mu       sync.Mutex
	payloads []map[string]any
}

func (p *PendingOpen) Set(payload map[string]any) {
	p.mu.Lock()
	p.payloads = append(p.payloads, payload)
	p.mu.Unlock()
}

func (p *PendingOpen) Take() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.payloads) == 0 {
		return nil
	}
	payload := p.payloads[0]
	p.payloads[0] = nil
	p.payloads = p.payloads[1:]
	return payload
}
