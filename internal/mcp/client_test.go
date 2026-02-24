// ABOUTME: Tests for MCP Client concurrency: notification handler bounded goroutines, context cancellation
// ABOUTME: Validates handleNotifications uses semaphore and respects client context

package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeTransport is a minimal Transport for testing Client behavior.
type fakeTransport struct {
	mu       sync.Mutex
	sendFunc func(ctx context.Context, req *Request) (*Response, error)
	incoming chan json.RawMessage
	closed   bool
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{
		incoming: make(chan json.RawMessage, 64),
	}
}

func (f *fakeTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	f.mu.Lock()
	fn := f.sendFunc
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return &Response{Result: json.RawMessage(`{}`)}, nil
}

func (f *fakeTransport) Notify(_ context.Context, _ *Notification) error {
	return nil
}

func (f *fakeTransport) Receive() <-chan json.RawMessage {
	return f.incoming
}

func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.incoming)
	}
	return nil
}

func TestClient_HandleNotificationsUsesTimeout(t *testing.T) {
	ft := newFakeTransport()

	var callCount atomic.Int32
	ft.sendFunc = func(ctx context.Context, req *Request) (*Response, error) {
		if req.Method == "tools/list" {
			// Verify the context has a deadline (timeout).
			if _, ok := ctx.Deadline(); !ok {
				t.Error("ListTools called without deadline; expected timeout context")
			}
			callCount.Add(1)
			return &Response{Result: json.RawMessage(`{"tools":[]}`)}, nil
		}
		return &Response{Result: json.RawMessage(`{}`)}, nil
	}

	c := NewClient(ft)
	err := c.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Send a tools/list_changed notification.
	notif := Notification{Method: "notifications/tools/list_changed"}
	data, _ := json.Marshal(notif)
	ft.incoming <- data

	// Give handleNotifications time to process.
	time.Sleep(200 * time.Millisecond)

	if callCount.Load() == 0 {
		t.Error("expected ListTools to be called on tools/list_changed notification")
	}

	_ = c.Close()
}

func TestClient_HandleNotificationsSemaphore(t *testing.T) {
	ft := newFakeTransport()

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	ft.sendFunc = func(ctx context.Context, req *Request) (*Response, error) {
		if req.Method == "tools/list" {
			cur := concurrent.Add(1)
			// Track max concurrent calls.
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
			return &Response{Result: json.RawMessage(`{"tools":[]}`)}, nil
		}
		return &Response{Result: json.RawMessage(`{}`)}, nil
	}

	c := NewClient(ft)
	err := c.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Flood with notifications.
	for range 10 {
		notif := Notification{Method: "notifications/tools/list_changed"}
		data, _ := json.Marshal(notif)
		ft.incoming <- data
	}

	time.Sleep(600 * time.Millisecond)

	if maxConcurrent.Load() > 1 {
		t.Errorf("max concurrent ListTools = %d; want <= 1 (semaphore should limit)", maxConcurrent.Load())
	}

	_ = c.Close()
}

func TestClient_CloseStopsNotificationHandler(t *testing.T) {
	ft := newFakeTransport()

	c := NewClient(ft)
	err := c.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Close should not hang; handleNotifications should exit when transport closes.
	done := make(chan struct{})
	go func() {
		_ = c.Close()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung; handleNotifications may be leaked")
	}
}
