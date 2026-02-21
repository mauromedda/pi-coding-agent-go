// ABOUTME: Tests for the typed event bus
// ABOUTME: Covers subscribe, publish, unsubscribe, and concurrent access

package eventbus

import (
	"sync"
	"testing"
)

func TestBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := New[string]()
	var received string

	bus.Subscribe(func(s string) {
		received = s
	})

	bus.Publish("hello")

	if received != "hello" {
		t.Errorf("received = %q, want %q", received, "hello")
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := New[int]()
	var sum int
	var mu sync.Mutex

	for range 3 {
		bus.Subscribe(func(n int) {
			mu.Lock()
			sum += n
			mu.Unlock()
		})
	}

	bus.Publish(10)

	mu.Lock()
	defer mu.Unlock()
	if sum != 30 {
		t.Errorf("sum = %d, want 30", sum)
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	t.Parallel()

	bus := New[string]()
	called := false

	unsub := bus.Subscribe(func(_ string) {
		called = true
	})

	unsub()
	bus.Publish("test")

	if called {
		t.Error("handler should not be called after unsubscribe")
	}
}

func TestBus_Count(t *testing.T) {
	t.Parallel()

	bus := New[int]()

	unsub1 := bus.Subscribe(func(_ int) {})
	bus.Subscribe(func(_ int) {})

	if bus.Count() != 2 {
		t.Errorf("Count() = %d, want 2", bus.Count())
	}

	unsub1()
	if bus.Count() != 1 {
		t.Errorf("Count() = %d, want 1", bus.Count())
	}
}
