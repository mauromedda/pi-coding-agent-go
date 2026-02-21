// ABOUTME: Tests for RecoverGoroutine panic recovery without os.Exit
// ABOUTME: Verifies goroutine panics are caught and terminal is restored

package terminal

import (
	"sync"
	"testing"
)

// mockTerminal records ExitRawMode calls for testing.
type mockTerminal struct {
	exitCalled bool
	mu         sync.Mutex
}

func (m *mockTerminal) EnterRawMode() error                  { return nil }
func (m *mockTerminal) ExitRawMode() error                   { m.mu.Lock(); m.exitCalled = true; m.mu.Unlock(); return nil }
func (m *mockTerminal) Size() (int, int, error)              { return 80, 24, nil }
func (m *mockTerminal) Write(p []byte) (int, error)          { return len(p), nil }
func (m *mockTerminal) OnResize(_ func(width, height int)) {}

func TestRecoverGoroutine_CatchesPanic(t *testing.T) {
	t.Parallel()

	mt := &mockTerminal{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer RecoverGoroutine(mt)
		panic("test goroutine panic")
	}()

	<-done

	mt.mu.Lock()
	defer mt.mu.Unlock()
	if !mt.exitCalled {
		t.Error("expected ExitRawMode to be called on goroutine panic")
	}
}

func TestRecoverGoroutine_NoPanic(t *testing.T) {
	t.Parallel()

	mt := &mockTerminal{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer RecoverGoroutine(mt)
		// no panic: normal return
	}()

	<-done

	mt.mu.Lock()
	defer mt.mu.Unlock()
	if mt.exitCalled {
		t.Error("ExitRawMode should not be called when no panic occurs")
	}
}
