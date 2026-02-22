// ABOUTME: Filterable scrollable list component with fuzzy matching
// ABOUTME: Supports arrow key navigation, viewport scrolling, and item selection

package component

import (
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/fuzzy"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// ListItem represents a single entry in the select list.
type ListItem struct {
	Label       string
	Description string
}

// SelectList is a filterable, scrollable list of items.
type SelectList struct {
	items     []ListItem
	visible   []ListItem
	selected  int
	scrollOff int
	maxHeight int
	filter    string
	dirty     bool
}

// NewSelectList creates a SelectList with the given items.
func NewSelectList(items []ListItem) *SelectList {
	sl := &SelectList{
		items:     items,
		maxHeight: 100,
		dirty:     true,
	}
	sl.applyFilter()
	return sl
}

// SetItems replaces the item list and resets selection.
func (sl *SelectList) SetItems(items []ListItem) {
	sl.items = items
	sl.selected = 0
	sl.scrollOff = 0
	sl.applyFilter()
	sl.dirty = true
}

// SetFilter sets the fuzzy filter string and refilters.
func (sl *SelectList) SetFilter(f string) {
	sl.filter = f
	sl.selected = 0
	sl.scrollOff = 0
	sl.applyFilter()
	sl.dirty = true
}

// SetMaxHeight limits the number of visible rows.
func (sl *SelectList) SetMaxHeight(h int) {
	sl.maxHeight = h
	sl.dirty = true
}

// SelectedIndex returns the index within the visible (filtered) items.
func (sl *SelectList) SelectedIndex() int {
	return sl.selected
}

// SelectedItem returns the currently selected item.
// Returns a zero-value ListItem if the list is empty.
func (sl *SelectList) SelectedItem() ListItem {
	if len(sl.visible) == 0 {
		return ListItem{}
	}
	return sl.visible[sl.selected]
}

// VisibleItems returns the currently filtered/visible items.
func (sl *SelectList) VisibleItems() []ListItem {
	return sl.visible
}

// Invalidate marks the component for re-render.
func (sl *SelectList) Invalidate() {
	sl.dirty = true
}

// HandleInput processes keyboard input for navigation.
func (sl *SelectList) HandleInput(data string) {
	k := key.ParseKey(data)
	switch k.Type {
	case key.KeyUp:
		sl.moveUp()
	case key.KeyDown:
		sl.moveDown()
	}
}

func (sl *SelectList) moveUp() {
	if sl.selected > 0 {
		sl.selected--
		sl.adjustScroll()
		sl.dirty = true
	}
}

func (sl *SelectList) moveDown() {
	if sl.selected < len(sl.visible)-1 {
		sl.selected++
		sl.adjustScroll()
		sl.dirty = true
	}
}

func (sl *SelectList) adjustScroll() {
	if sl.selected < sl.scrollOff {
		sl.scrollOff = sl.selected
	}
	if sl.selected >= sl.scrollOff+sl.maxHeight {
		sl.scrollOff = sl.selected - sl.maxHeight + 1
	}
}

func (sl *SelectList) applyFilter() {
	if sl.filter == "" {
		sl.visible = make([]ListItem, len(sl.items))
		copy(sl.visible, sl.items)
		return
	}

	labels := make([]string, len(sl.items))
	for i, item := range sl.items {
		labels[i] = item.Label
	}
	matches := fuzzy.Find(sl.filter, labels)
	sl.visible = make([]ListItem, len(matches))
	for i, m := range matches {
		sl.visible[i] = sl.items[m.Index]
	}
}

// Render writes the list into the buffer.
func (sl *SelectList) Render(out *tui.RenderBuffer, w int) {
	if len(sl.visible) == 0 {
		return
	}

	end := min(sl.scrollOff+sl.maxHeight, len(sl.visible))

	for i := sl.scrollOff; i < end; i++ {
		item := sl.visible[i]
		line := sl.formatItem(item, w, i == sl.selected)
		out.WriteLine(line)
	}
}

func (sl *SelectList) formatItem(item ListItem, w int, selected bool) string {
	p := theme.Current().Palette
	label := item.Label
	desc := item.Description

	var line string
	if desc != "" {
		line = fmt.Sprintf("  %s  %s%s\x1b[0m", label, p.Muted.Code(), desc)
	} else {
		line = fmt.Sprintf("  %s", label)
	}

	line = width.TruncateToWidth(line, w)

	if selected {
		line = p.Bold.Code() + p.Selection.Code() + line + "\x1b[0m"
	}
	return line
}
