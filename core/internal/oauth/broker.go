package oauth

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type CallbackPayload struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

type CallbackBroker struct {
	mu      sync.Mutex
	pending map[string]chan CallbackPayload
}

func NewCallbackBroker() *CallbackBroker {
	return &CallbackBroker{pending: make(map[string]chan CallbackPayload)}
}

func (b *CallbackBroker) Register(state string) <-chan CallbackPayload {
	ch := make(chan CallbackPayload, 1)
	b.mu.Lock()
	b.pending[state] = ch
	b.mu.Unlock()
	return ch
}

func (b *CallbackBroker) Unregister(state string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.pending[state]; ok {
		close(ch)
		delete(b.pending, state)
	}
}

func (b *CallbackBroker) Deliver(state string, payload CallbackPayload) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch, ok := b.pending[state]
	if !ok {
		return false
	}
	select {
	case ch <- payload:
	default:
	}
	delete(b.pending, state)
	close(ch)
	return true
}

type BrokerFlow struct {
	state       string
	verifier    string
	redirectURL string
	broker      *CallbackBroker
	resultCh    <-chan CallbackPayload
}

func StartBrokerFlow(broker *CallbackBroker, redirectURL string) (*BrokerFlow, error) {
	switch {
	case broker == nil:
		return nil, errors.New("oauth: broker is required")
	case redirectURL == "":
		return nil, errors.New("oauth: redirect url is required")
	}

	state, err := randomToken(32)
	if err != nil {
		return nil, err
	}

	flow := &BrokerFlow{
		state:       state,
		verifier:    oauth2.GenerateVerifier(),
		redirectURL: redirectURL,
		broker:      broker,
	}
	flow.resultCh = broker.Register(state)
	return flow, nil
}

func (f *BrokerFlow) State() string       { return f.state }
func (f *BrokerFlow) Verifier() string    { return f.verifier }
func (f *BrokerFlow) RedirectURL() string { return f.redirectURL }

func (f *BrokerFlow) Wait(ctx context.Context, timeout time.Duration) (CallbackResult, error) {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case payload, ok := <-f.resultCh:
		if !ok {
			return CallbackResult{}, errors.New("oauth: callback channel closed")
		}
		if payload.Error != "" {
			return CallbackResult{}, errors.New(FormatCallbackError(payload.Error, payload.ErrorDescription))
		}
		return CallbackResult{Code: payload.Code, State: payload.State}, nil
	case <-ctx.Done():
		f.broker.Unregister(f.state)
		return CallbackResult{}, ctx.Err()
	case <-timer.C:
		f.broker.Unregister(f.state)
		return CallbackResult{}, errors.New("oauth callback timed out")
	}
}

func (f *BrokerFlow) Close() error {
	f.broker.Unregister(f.state)
	return nil
}
