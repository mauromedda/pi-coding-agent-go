// ABOUTME: Tests for lock-free global theme pointer: Current, Set, concurrent access
// ABOUTME: Verifies atomic swap semantics and default theme initialization

package theme

import (
	"sync"
	"testing"
)

func TestCurrent_ReturnsDefault(t *testing.T) {
	t.Parallel()
	th := Current()
	if th == nil {
		t.Fatal("Current() returned nil")
	}
	if th.Name != "default" {
		t.Errorf("Current().Name = %q; want %q", th.Name, "default")
	}
}

func TestSet_ChangesCurrent(t *testing.T) {
	t.Parallel()
	custom := &Theme{Name: "custom", Palette: DefaultPalette()}
	old := Current()
	Set(custom)
	defer Set(old) // restore

	got := Current()
	if got.Name != "custom" {
		t.Errorf("after Set(), Current().Name = %q; want %q", got.Name, "custom")
	}
}

func TestCurrent_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			_ = Current()
		})
	}
	wg.Wait()
}
