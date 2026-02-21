// ABOUTME: Tests for StdinBuffer key reading and dispatch from an io.Reader.
// ABOUTME: Uses bytes.Buffer for deterministic input; covers single keys, escape sequences, and context cancellation.

package input

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

func TestStdinBuffer_SingleKey(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("a")
	var mu sync.Mutex
	var got []key.Key

	buf := NewStdinBuffer(input, func(k key.Key) {
		mu.Lock()
		got = append(got, k)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	buf.Start(ctx)

	mu.Lock()
	defer mu.Unlock()

	if len(got) == 0 {
		t.Fatal("expected at least one key, got none")
	}
	if got[0].Type != key.KeyRune || got[0].Rune != 'a' {
		t.Errorf("expected KeyRune 'a', got Type=%v Rune=%q", got[0].Type, got[0].Rune)
	}
}

func TestStdinBuffer_EscapeSequence(t *testing.T) {
	t.Parallel()

	// Send an arrow-up escape sequence in one chunk
	input := bytes.NewBufferString("\x1b[A")
	var mu sync.Mutex
	var got []key.Key

	buf := NewStdinBuffer(input, func(k key.Key) {
		mu.Lock()
		got = append(got, k)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	buf.Start(ctx)

	mu.Lock()
	defer mu.Unlock()

	if len(got) == 0 {
		t.Fatal("expected at least one key, got none")
	}
	if got[0].Type != key.KeyUp {
		t.Errorf("expected KeyUp, got Type=%v", got[0].Type)
	}
}

func TestStdinBuffer_MultipleKeys(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("abc")
	var mu sync.Mutex
	var got []key.Key

	buf := NewStdinBuffer(input, func(k key.Key) {
		mu.Lock()
		got = append(got, k)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	buf.Start(ctx)

	mu.Lock()
	defer mu.Unlock()

	if len(got) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(got))
	}
	for i, expected := range []rune{'a', 'b', 'c'} {
		if got[i].Type != key.KeyRune || got[i].Rune != expected {
			t.Errorf("key[%d]: expected KeyRune %q, got Type=%v Rune=%q", i, expected, got[i].Type, got[i].Rune)
		}
	}
}

func TestStdinBuffer_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Use a reader that blocks forever; context cancellation must stop Start.
	r, _ := syncPipe()
	var got []key.Key

	buf := NewStdinBuffer(r, func(k key.Key) {
		got = append(got, k)
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		buf.Start(ctx)
		close(done)
	}()

	// Cancel immediately
	cancel()

	select {
	case <-done:
		// Start returned; success
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

// syncPipe creates a pipe-like reader/writer pair for testing.
// The reader blocks until data is written or the writer is closed.
func syncPipe() (*blockingReader, chan<- []byte) {
	ch := make(chan []byte)
	return &blockingReader{ch: ch}, ch
}

// blockingReader reads from a channel, blocking until data arrives.
type blockingReader struct {
	ch  <-chan []byte
	buf []byte
}

func (r *blockingReader) Read(p []byte) (int, error) {
	if len(r.buf) > 0 {
		n := copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}
	data, ok := <-r.ch
	if !ok {
		return 0, context.Canceled
	}
	n := copy(p, data)
	if n < len(data) {
		r.buf = data[n:]
	}
	return n, nil
}
