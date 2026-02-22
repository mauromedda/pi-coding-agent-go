// ABOUTME: CommandPalette overlay for slash-command autocomplete
// ABOUTME: Fuzzy-filters commands, highlights selection, caps visible items

package components

import (
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

const maxVisibleItems = 10

// CommandEntry describes a single slash command for the palette.
type CommandEntry struct {
	Name        string
	Description string
}

// CommandPalette is a filterable overlay listing available slash commands.
type CommandPalette struct {
	commands []CommandEntry
	visible  []CommandEntry
	selected int
	filter   string
}

// NewCommandPalette creates a palette pre-populated with the given commands.
func NewCommandPalette(cmds []CommandEntry) *CommandPalette {
	cp := &CommandPalette{
		commands: cmds,
	}
	cp.applyFilter()
	return cp
}

// SetFilter updates the fuzzy-match filter string and resets selection.
func (cp *CommandPalette) SetFilter(f string) {
	cp.filter = f
	cp.selected = 0
	cp.applyFilter()
}

// Selected returns the Name of the currently highlighted command.
func (cp *CommandPalette) Selected() string {
	if len(cp.visible) == 0 {
		return ""
	}
	return cp.visible[cp.selected].Name
}

// MoveDown advances the selection, wrapping at the end.
func (cp *CommandPalette) MoveDown() {
	if len(cp.visible) == 0 {
		return
	}
	cp.selected = (cp.selected + 1) % len(cp.visible)
}

// MoveUp moves the selection up, wrapping at the top.
func (cp *CommandPalette) MoveUp() {
	if len(cp.visible) == 0 {
		return
	}
	cp.selected = (cp.selected - 1 + len(cp.visible)) % len(cp.visible)
}

// VisibleCount returns the number of commands passing the current filter.
func (cp *CommandPalette) VisibleCount() int {
	return len(cp.visible)
}

// Render writes the command list into out, capped at maxVisibleItems.
// The viewport scrolls to keep the selected item visible.
func (cp *CommandPalette) Render(out *tui.RenderBuffer, w int) {
	total := len(cp.visible)
	if total == 0 {
		return
	}

	// Compute viewport window around selected item
	start := 0
	end := total
	if total > maxVisibleItems {
		// Ensure selected item is within the visible window
		start = cp.selected - maxVisibleItems/2
		if start < 0 {
			start = 0
		}
		end = start + maxVisibleItems
		if end > total {
			end = total
			start = end - maxVisibleItems
		}
	}

	p := theme.Current().Palette
	for i := start; i < end; i++ {
		entry := cp.visible[i]
		name := fmt.Sprintf("/%s", entry.Name)
		desc := entry.Description

		// Truncate if too wide: "  /name   description"
		line := fmt.Sprintf("  %-16s %s", name, desc)
		line = width.TruncateToWidth(line, w)

		if i == cp.selected {
			line = p.Bold.Code() + p.Selection.Code() + line + "\x1b[0m" // bold + inverse
		} else {
			line = p.Dim.Code() + line + "\x1b[0m" // dim
		}
		out.WriteLine(line)
	}
}

// Invalidate is a no-op; the palette re-renders every frame.
func (cp *CommandPalette) Invalidate() {}

// applyFilter rebuilds the visible list from the current filter.
func (cp *CommandPalette) applyFilter() {
	if cp.filter == "" {
		cp.visible = make([]CommandEntry, len(cp.commands))
		copy(cp.visible, cp.commands)
		return
	}

	lower := strings.ToLower(cp.filter)
	cp.visible = cp.visible[:0]
	for _, cmd := range cp.commands {
		if strings.Contains(strings.ToLower(cmd.Name), lower) {
			cp.visible = append(cp.visible, cmd)
		}
	}
}
