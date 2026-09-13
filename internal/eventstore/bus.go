package eventstore

import (
	"sync"
)

// Handler is invoked for every published envelope matching its subscription.
type Handler func(env Envelope)

// Bus is an in-process, in-memory event bus that dispatches published
// envelopes to subscribed handlers synchronously in publish order.
//
// It is safe for concurrent use. Handlers are invoked under the bus lock,
// so handlers must not publish back into the same bus to avoid deadlock.
type Bus struct {
	mu       sync.Mutex
	handlers []handler
	closed   bool
}

type handler struct {
	match  func(Envelope) bool
	handle Handler
}

// NewBus creates a new in-memory event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a handler for envelopes matching the optional matcher.
// If match is nil, the handler receives all envelopes.
func (b *Bus) Subscribe(match func(Envelope) bool, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if match == nil {
		match = func(Envelope) bool { return true }
	}
	b.handlers = append(b.handlers, handler{match: match, handle: h})
}

// Publish dispatches the envelope to all matching handlers in subscription
// order.
func (b *Bus) Publish(env Envelope) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	handlers := make([]handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.Unlock()

	for _, h := range handlers {
		if h.match(env) {
			h.handle(env)
		}
	}
}

// Close stops dispatching further published envelopes.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}
