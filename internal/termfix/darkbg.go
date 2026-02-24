// ABOUTME: Pre-sets lipgloss dark background before BubbleTea's init() sends OSC queries
// ABOUTME: Must be imported (with _) before any package that imports bubbletea

package termfix

import "github.com/charmbracelet/lipgloss"

func init() {
	// Tell lipgloss we have a dark background so it never sends
	// OSC 10/11 terminal queries. BubbleTea's own init() calls
	// lipgloss.HasDarkBackground(); if explicitBackgroundColor is
	// already set, the sync.Once that fires the query is skipped.
	//
	// This package must NOT import bubbletea (directly or transitively)
	// so that Go's init order guarantees this runs first.
	lipgloss.SetHasDarkBackground(true)
}
