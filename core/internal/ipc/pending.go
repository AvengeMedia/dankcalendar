package ipc

import "sync"

// PendingOpen holds a "ui" action captured before the GUI subscribed to the
// "ui" topic (e.g. a webcal:// link or an event open that cold-started the
// app). It is flushed onto the bus the moment a "ui" subscriber appears.
type PendingOpen struct {
	mu      sync.Mutex
	payload map[string]any
}

func (p *PendingOpen) Set(payload map[string]any) {
	p.mu.Lock()
	p.payload = payload
	p.mu.Unlock()
}

func (p *PendingOpen) Take() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	payload := p.payload
	p.payload = nil
	return payload
}
