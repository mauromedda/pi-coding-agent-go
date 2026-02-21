// ABOUTME: Typed event bus with subscriber management for decoupled components
// ABOUTME: Supports subscribe/unsubscribe with goroutine-safe delivery

package eventbus

import "sync"

// Handler is a callback function for events.
type Handler[T any] func(T)

// Bus is a typed event bus that delivers events to registered handlers.
type Bus[T any] struct {
	mu       sync.RWMutex
	handlers map[int]Handler[T]
	nextID   int
}

// New creates a new event bus.
func New[T any]() *Bus[T] {
	return &Bus[T]{
		handlers: make(map[int]Handler[T]),
	}
}

// Subscribe registers a handler and returns an unsubscribe function.
func (b *Bus[T]) Subscribe(handler Handler[T]) func() {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.handlers[id] = handler
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		delete(b.handlers, id)
		b.mu.Unlock()
	}
}

// Publish sends an event to all registered handlers.
// Handlers are called synchronously in arbitrary order.
func (b *Bus[T]) Publish(event T) {
	b.mu.RLock()
	// Snapshot handlers to avoid holding lock during callbacks
	snapshot := make([]Handler[T], 0, len(b.handlers))
	for _, h := range b.handlers {
		snapshot = append(snapshot, h)
	}
	b.mu.RUnlock()

	for _, h := range snapshot {
		h(event)
	}
}

// Count returns the number of registered handlers.
func (b *Bus[T]) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers)
}
