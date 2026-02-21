// ABOUTME: Searchable prompt history component with reverse search and file persistence
// ABOUTME: Supports Ctrl+R search mode, up/down navigation, and load/save to ~/.pi-go/history

package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

// History manages a searchable prompt history with persistence.
type History struct {
	entries   []string
	pos       int // -1 means "no selection" (at the input line)
	focused   bool
	searching bool
	query     string
	matchIdx  int // index within entries of current search match
	dirty     bool
}

// NewHistory creates a new empty History component.
func NewHistory() *History {
	return &History{
		entries: make([]string, 0, 256),
		pos:     -1,
		dirty:   true,
	}
}

// Len returns the number of history entries.
func (h *History) Len() int {
	return len(h.entries)
}

// Add appends a new entry to the history, deduplicating consecutive duplicates.
func (h *History) Add(entry string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}
	// Remove existing duplicate
	for i, e := range h.entries {
		if e == entry {
			h.entries = append(h.entries[:i], h.entries[i+1:]...)
			break
		}
	}
	h.entries = append(h.entries, entry)
	h.pos = -1
	h.dirty = true
}

// Current returns the currently selected history entry, or "" if none.
func (h *History) Current() string {
	if h.searching {
		if h.matchIdx >= 0 && h.matchIdx < len(h.entries) {
			return h.entries[h.matchIdx]
		}
		return ""
	}
	if h.pos < 0 || h.pos >= len(h.entries) {
		return ""
	}
	// pos 0 = most recent (last in entries slice)
	idx := len(h.entries) - 1 - h.pos
	if idx < 0 || idx >= len(h.entries) {
		return ""
	}
	return h.entries[idx]
}

// Reset returns the navigation position to "no selection".
func (h *History) Reset() {
	h.pos = -1
	h.searching = false
	h.query = ""
	h.matchIdx = -1
	h.dirty = true
}

// StartSearch enters reverse search mode.
func (h *History) StartSearch() {
	h.searching = true
	h.query = ""
	h.matchIdx = -1
	h.dirty = true
}

// IsSearching returns true if reverse search mode is active.
func (h *History) IsSearching() bool {
	return h.searching
}

// SetFocused sets the focus state.
func (h *History) SetFocused(focused bool) {
	h.focused = focused
	h.dirty = true
}

// IsFocused returns the focus state.
func (h *History) IsFocused() bool {
	return h.focused
}

// Invalidate marks the component for re-render.
func (h *History) Invalidate() {
	h.dirty = true
}

// HandleInput processes keyboard input.
func (h *History) HandleInput(data string) {
	if h.searching {
		h.handleSearchInput(data)
		return
	}

	k := key.ParseKey(data)
	switch k.Type {
	case key.KeyUp:
		h.navigateUp()
	case key.KeyDown:
		h.navigateDown()
	case key.KeyCtrlR:
		h.StartSearch()
	}
}

func (h *History) handleSearchInput(data string) {
	k := key.ParseKey(data)

	switch k.Type {
	case key.KeyEscape:
		h.searching = false
		h.query = ""
		h.matchIdx = -1
		h.dirty = true
	case key.KeyEnter:
		// Accept current match
		h.searching = false
		h.dirty = true
	case key.KeyBackspace:
		if len(h.query) > 0 {
			h.query = h.query[:len(h.query)-1]
			h.updateSearch()
		}
	case key.KeyRune:
		h.query += string(k.Rune)
		h.updateSearch()
	case key.KeyCtrlR:
		// Search for next older match
		h.searchNextOlder()
	}
}

func (h *History) updateSearch() {
	h.matchIdx = -1
	if h.query == "" {
		h.dirty = true
		return
	}
	// Search from most recent to oldest
	for i := len(h.entries) - 1; i >= 0; i-- {
		if strings.Contains(h.entries[i], h.query) {
			h.matchIdx = i
			h.dirty = true
			return
		}
	}
	h.dirty = true
}

func (h *History) searchNextOlder() {
	if h.query == "" {
		return
	}
	start := h.matchIdx - 1
	if start < 0 {
		start = len(h.entries) - 1
	}
	for i := start; i >= 0; i-- {
		if strings.Contains(h.entries[i], h.query) {
			h.matchIdx = i
			h.dirty = true
			return
		}
	}
}

func (h *History) navigateUp() {
	if len(h.entries) == 0 {
		return
	}
	if h.pos < len(h.entries)-1 {
		h.pos++
		h.dirty = true
	}
}

func (h *History) navigateDown() {
	if h.pos > -1 {
		h.pos--
		h.dirty = true
	}
}

// Render writes the history display into the buffer.
func (h *History) Render(out *tui.RenderBuffer, w int) {
	if h.searching {
		prompt := fmt.Sprintf("(reverse-i-search)`%s': ", h.query)
		match := h.Current()
		line := prompt + match
		if len(line) > w && w > 0 {
			line = line[:w]
		}
		out.WriteLine(line)
		return
	}
	// When not searching, history renders nothing visible by itself;
	// the parent component reads Current() for display.
}

// SaveToFile writes history entries to the given file path, one per line.
func (h *History) SaveToFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	content := strings.Join(h.entries, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing history file: %w", err)
	}
	return nil
}

// LoadFromFile reads history entries from the given file path.
// Returns nil if the file does not exist (fresh start).
func (h *History) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading history file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	h.entries = h.entries[:0]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			h.entries = append(h.entries, trimmed)
		}
	}
	h.pos = -1
	h.dirty = true
	return nil
}
