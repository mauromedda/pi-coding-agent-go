// ABOUTME: Tests for VirtualTerminal verifying raw mode tracking, output capture, and resize.
// ABOUTME: Uses table-driven and parallel sub-tests for thorough coverage.

package terminal

import (
	"sync"
	"testing"
)

// compile-time check: VirtualTerminal must satisfy Terminal.
var _ Terminal = (*VirtualTerminal)(nil)

func TestVirtualTerminal_Size(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{name: "standard 80x24", width: 80, height: 24, wantWidth: 80, wantHeight: 24},
		{name: "wide 200x50", width: 200, height: 50, wantWidth: 200, wantHeight: 50},
		{name: "zero dimensions", width: 0, height: 0, wantWidth: 0, wantHeight: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vt := NewVirtualTerminal(tt.width, tt.height)

			w, h, err := vt.Size()
			if err != nil {
				t.Fatalf("Size() unexpected error: %v", err)
			}
			if w != tt.wantWidth || h != tt.wantHeight {
				t.Errorf("Size() = (%d, %d), want (%d, %d)", w, h, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestVirtualTerminal_RawMode(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	if vt.IsRawMode() {
		t.Fatal("expected raw mode to be off initially")
	}

	if err := vt.EnterRawMode(); err != nil {
		t.Fatalf("EnterRawMode() unexpected error: %v", err)
	}
	if !vt.IsRawMode() {
		t.Fatal("expected raw mode to be on after EnterRawMode")
	}
	if vt.EnterCount() != 1 {
		t.Errorf("EnterCount() = %d, want 1", vt.EnterCount())
	}

	if err := vt.ExitRawMode(); err != nil {
		t.Fatalf("ExitRawMode() unexpected error: %v", err)
	}
	if vt.IsRawMode() {
		t.Fatal("expected raw mode to be off after ExitRawMode")
	}
	if vt.ExitCount() != 1 {
		t.Errorf("ExitCount() = %d, want 1", vt.ExitCount())
	}
}

func TestVirtualTerminal_MultipleRawModeTransitions(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	for i := range 3 {
		if err := vt.EnterRawMode(); err != nil {
			t.Fatalf("iteration %d: EnterRawMode() error: %v", i, err)
		}
		if err := vt.ExitRawMode(); err != nil {
			t.Fatalf("iteration %d: ExitRawMode() error: %v", i, err)
		}
	}

	if vt.EnterCount() != 3 {
		t.Errorf("EnterCount() = %d, want 3", vt.EnterCount())
	}
	if vt.ExitCount() != 3 {
		t.Errorf("ExitCount() = %d, want 3", vt.ExitCount())
	}
}

func TestVirtualTerminal_Write(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	data := []byte("hello, terminal")
	n, err := vt.Write(data)
	if err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() returned n=%d, want %d", n, len(data))
	}
	if got := vt.Output(); got != "hello, terminal" {
		t.Errorf("Output() = %q, want %q", got, "hello, terminal")
	}
}

func TestVirtualTerminal_WriteAccumulates(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	if _, err := vt.Write([]byte("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := vt.Write([]byte("two")); err != nil {
		t.Fatal(err)
	}

	if got := vt.Output(); got != "onetwo" {
		t.Errorf("Output() = %q, want %q", got, "onetwo")
	}
}

func TestVirtualTerminal_Reset(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	if _, err := vt.Write([]byte("some data")); err != nil {
		t.Fatal(err)
	}
	vt.Reset()

	if got := vt.Output(); got != "" {
		t.Errorf("Output() after Reset = %q, want empty", got)
	}
}

func TestVirtualTerminal_OnResize(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	var gotWidth, gotHeight int
	vt.OnResize(func(w, h int) {
		gotWidth = w
		gotHeight = h
	})

	vt.SetSize(120, 40)

	if gotWidth != 120 || gotHeight != 40 {
		t.Errorf("resize callback got (%d, %d), want (120, 40)", gotWidth, gotHeight)
	}

	// Size should also reflect the new dimensions.
	w, h, err := vt.Size()
	if err != nil {
		t.Fatalf("Size() unexpected error: %v", err)
	}
	if w != 120 || h != 40 {
		t.Errorf("Size() after SetSize = (%d, %d), want (120, 40)", w, h)
	}
}

func TestVirtualTerminal_SetSizeWithoutCallback(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	// Should not panic when no callback is registered.
	vt.SetSize(100, 50)

	w, h, err := vt.Size()
	if err != nil {
		t.Fatalf("Size() unexpected error: %v", err)
	}
	if w != 100 || h != 50 {
		t.Errorf("Size() = (%d, %d), want (100, 50)", w, h)
	}
}

func TestVirtualTerminal_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	vt := NewVirtualTerminal(80, 24)

	var wg sync.WaitGroup
	const goroutines = 10

	// Concurrent writes.
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = vt.Write([]byte("x"))
		}()
	}

	// Concurrent size reads.
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_, _, _ = vt.Size()
		}()
	}

	// Concurrent raw mode toggles.
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = vt.EnterRawMode()
			_ = vt.ExitRawMode()
		}()
	}

	wg.Wait()

	// We only verify no data race; output length may vary.
	if len(vt.Output()) != goroutines {
		t.Errorf("Output length = %d, want %d", len(vt.Output()), goroutines)
	}
}

func TestVirtualTerminal_ImplementsTerminal(t *testing.T) {
	t.Parallel()

	// This is a compile-time check via the var above, but let's also
	// exercise the interface assignment at runtime for clarity.
	var term Terminal = NewVirtualTerminal(80, 24)
	if _, _, err := term.Size(); err != nil {
		t.Fatalf("Terminal.Size() unexpected error: %v", err)
	}
}
