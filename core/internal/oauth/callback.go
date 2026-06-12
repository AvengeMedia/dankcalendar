package oauth

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Callback interface {
	State() string
	Verifier() string
	RedirectURL() string
	Wait(ctx context.Context, timeout time.Duration) (CallbackResult, error)
	Close() error
}

type AuthFlow interface {
	State() string
	Close() error
}

type FlowRegistry struct {
	mu    sync.Mutex
	flows map[string]AuthFlow
}

func NewFlowRegistry() *FlowRegistry {
	return &FlowRegistry{flows: make(map[string]AuthFlow)}
}

func (r *FlowRegistry) Register(flow AuthFlow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flows[flow.State()] = flow
}

func (r *FlowRegistry) Take(state string) (AuthFlow, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	flow, ok := r.flows[state]
	if !ok {
		return nil, false
	}
	delete(r.flows, state)
	return flow, true
}

func (r *FlowRegistry) Cancel(state string) bool {
	r.mu.Lock()
	flow, ok := r.flows[state]
	if ok {
		delete(r.flows, state)
	}
	r.mu.Unlock()

	if !ok {
		return false
	}
	_ = flow.Close()
	return true
}

func (r *FlowRegistry) CloseAll() {
	r.mu.Lock()
	flows := r.flows
	r.flows = make(map[string]AuthFlow)
	r.mu.Unlock()

	for _, flow := range flows {
		_ = flow.Close()
	}
}

var ErrFlowNotFound = errors.New("oauth: flow not found")
