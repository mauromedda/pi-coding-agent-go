// ABOUTME: Tests for HTTP transport close race and trySendIncoming safety
// ABOUTME: Validates that Close + trySendIncoming cannot panic on closed channel

package mcp

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestHTTPTransport_TrySendIncomingAfterClose(t *testing.T) {
	// After Close, trySendIncoming must not panic.
	// The fix removes close(t.incoming) from Close; consumers detect
	// completion via the done channel.
	t.Parallel()

	// We cannot use NewHTTPTransport (it starts SSE listener to a real URL),
	// so construct a minimal instance for the trySendIncoming path.
	tr := &HTTPTransport{
		incoming: make(chan json.RawMessage, 64),
		done:     make(chan struct{}),
	}

	close(tr.done) // simulate Close

	// This must not panic even though done is closed.
	tr.trySendIncoming(json.RawMessage(`{"test": true}`))
}

func TestHTTPTransport_ConcurrentTrySendAndClose(t *testing.T) {
	t.Parallel()

	for range 100 {
		tr := &HTTPTransport{
			incoming: make(chan json.RawMessage, 8),
			done:     make(chan struct{}),
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for range 20 {
				tr.trySendIncoming(json.RawMessage(`{}`))
			}
		}()

		go func() {
			defer wg.Done()
			close(tr.done) // simulate Close closing done channel
		}()

		wg.Wait()
	}
}
