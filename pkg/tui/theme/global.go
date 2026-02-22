// ABOUTME: Lock-free global theme pointer using atomic.Pointer
// ABOUTME: Current() returns the active theme; Set() swaps it atomically

package theme

import "sync/atomic"

var current atomic.Pointer[Theme]

func init() {
	p := DefaultPalette()
	current.Store(&Theme{Name: "default", Palette: p})
}

// Current returns the active theme. Never returns nil.
func Current() *Theme {
	return current.Load()
}

// Set atomically replaces the active theme.
func Set(t *Theme) {
	current.Store(t)
}
