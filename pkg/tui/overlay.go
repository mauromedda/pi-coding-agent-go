// ABOUTME: Overlay types for modal dialogs rendered on top of main content
// ABOUTME: Supports centered, top-anchored, and bottom-anchored positioning

package tui

// OverlayPosition defines where an overlay is rendered.
type OverlayPosition int

const (
	OverlayCenter OverlayPosition = iota
	OverlayTop
	OverlayBottom
)

// Overlay represents a modal component rendered on top of the main container.
type Overlay struct {
	Component Component
	Position  OverlayPosition
	Width     int // 0 means use terminal width
	Height    int // 0 means auto-size from render output
}
