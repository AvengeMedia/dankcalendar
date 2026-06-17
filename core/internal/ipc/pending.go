package ipc

import "sync"

// PendingOpen holds a subscription URL captured before the GUI has subscribed
// to the "ui" topic (e.g. a webcal:// link that cold-started the app). It is
// flushed onto the bus the moment a "ui" subscriber appears.
type PendingOpen struct {
	mu  sync.Mutex
	url string
}

func (p *PendingOpen) Set(url string) {
	p.mu.Lock()
	p.url = url
	p.mu.Unlock()
}

func (p *PendingOpen) Take() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	url := p.url
	p.url = ""
	return url
}
