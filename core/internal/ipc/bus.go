package ipc

import (
	"context"
	"sync"
)

type Event struct {
	Topic string
	Data  any
}

type Subscriber struct {
	bus      *EventBus
	ctx      context.Context
	writer   *ConnWriter
	mu       sync.Mutex
	topics   map[string]bool
	closed   bool
	cancelFn context.CancelFunc
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[*Subscriber]struct{}
}

func NewEventBus() *EventBus {
	return &EventBus{subscribers: make(map[*Subscriber]struct{})}
}

func (b *EventBus) Publish(topic string, data any) {
	b.mu.RLock()
	subs := make([]*Subscriber, 0, len(b.subscribers))
	for s := range b.subscribers {
		subs = append(subs, s)
	}
	b.mu.RUnlock()

	for _, s := range subs {
		s.deliver(Event{Topic: topic, Data: data})
	}
}

func (b *EventBus) HasSubscriber(topic string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.subscribers {
		s.mu.Lock()
		subscribed := !s.closed && s.topics[topic]
		s.mu.Unlock()
		if subscribed {
			return true
		}
	}
	return false
}

func (b *EventBus) NewSubscriber(ctx context.Context, writer *ConnWriter) *Subscriber {
	cctx, cancel := context.WithCancel(ctx)
	s := &Subscriber{
		bus:      b,
		ctx:      cctx,
		writer:   writer,
		topics:   make(map[string]bool),
		cancelFn: cancel,
	}

	b.mu.Lock()
	b.subscribers[s] = struct{}{}
	b.mu.Unlock()
	return s
}

func (s *Subscriber) Subscribe(topics ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range topics {
		s.topics[t] = true
	}
}

func (s *Subscriber) Unsubscribe(topics ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range topics {
		delete(s.topics, t)
	}
}

func (s *Subscriber) Topics() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.topics))
	for t := range s.topics {
		out = append(out, t)
	}
	return out
}

func (s *Subscriber) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	s.cancelFn()
	s.bus.mu.Lock()
	delete(s.bus.subscribers, s)
	s.bus.mu.Unlock()
}

func (s *Subscriber) deliver(ev Event) {
	s.mu.Lock()
	if s.closed || !s.topics[ev.Topic] {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.writer.WriteEvent(ev)
}
