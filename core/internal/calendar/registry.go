package calendar

import (
	"context"
	"fmt"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	factories map[AccountKind]ProviderFactory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[AccountKind]ProviderFactory)}
}

func (r *Registry) Register(f ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[f.Kind()] = f
}

func (r *Registry) Build(ctx context.Context, acc Account, secrets SecretStore) (Provider, error) {
	r.mu.RLock()
	f, ok := r.factories[acc.Kind]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no provider registered for account kind %q", acc.Kind)
	}
	return f.Build(ctx, acc, secrets)
}

func (r *Registry) Kinds() []AccountKind {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]AccountKind, 0, len(r.factories))
	for k := range r.factories {
		out = append(out, k)
	}
	return out
}
