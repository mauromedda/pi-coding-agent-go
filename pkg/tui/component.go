// ABOUTME: Core TUI interfaces: Component, InputHandler, Focusable
// ABOUTME: Defines the contract for all renderable UI elements

package tui

// CursorMarker is a zero-width marker that components embed in render output
// to indicate cursor position. The TUI engine strips it and positions the
// real terminal cursor at that location.
const CursorMarker = "\x1b_pi:c\x07"

// Component is the base interface for all TUI elements.
// Components render into a pooled RenderBuffer and must not exceed the given width.
type Component interface {
	// Render writes the component's visual lines into out.
	// Lines must not exceed width visible columns.
	Render(out *RenderBuffer, width int)

	// Invalidate clears any cached render state, forcing a full re-render
	// on the next Render call.
	Invalidate()
}

// InputHandler is implemented by components that process keyboard input.
type InputHandler interface {
	HandleInput(data string)
}

// Focusable is implemented by components that participate in focus management.
type Focusable interface {
	SetFocused(focused bool)
	IsFocused() bool
}
