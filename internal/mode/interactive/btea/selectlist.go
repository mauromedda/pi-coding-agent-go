// ABOUTME: SelectListModel is a Bubble Tea leaf for filterable scrollable lists
// ABOUTME: Port of pkg/tui/component/selectlist.go; uses fuzzy.Find for filtering

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/fuzzy"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// ListItem represents a single entry in the select list.
type ListItem struct {
	Label       string
	Description string
}

// SelectListModel is a filterable, scrollable list of items.
// Implements tea.Model with value semantics (no mutex needed).
type SelectListModel struct {
	items     []ListItem
	visible   []ListItem
	selected  int
	scrollOff int
	maxHeight int
	filter    string
	width     int
}

// NewSelectListModel creates a SelectListModel with the given items.
func NewSelectListModel(items []ListItem) SelectListModel {
	m := SelectListModel{
		items:     items,
		maxHeight: 100,
	}
	m.applyFilter()
	return m
}

// Init returns nil; no commands needed at startup.
func (m SelectListModel) Init() tea.Cmd {
	return nil
}

// Update handles key and window-size messages.
func (m SelectListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the visible items with scrolling viewport.
func (m SelectListModel) View() string {
	if len(m.visible) == 0 {
		return ""
	}

	s := Styles()
	end := min(m.scrollOff+m.maxHeight, len(m.visible))
	var b strings.Builder

	for i := m.scrollOff; i < end; i++ {
		item := m.visible[i]
		selected := i == m.selected
		line := formatListItem(s, item, m.width, selected)
		if i > m.scrollOff {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}

	return b.String()
}

// SetFilter sets the fuzzy filter string and refilters. Returns a new model.
func (m SelectListModel) SetFilter(f string) SelectListModel {
	m.filter = f
	m.selected = 0
	m.scrollOff = 0
	m.applyFilter()
	return m
}

// SetItems replaces the item list and resets selection. Returns a new model.
func (m SelectListModel) SetItems(items []ListItem) SelectListModel {
	m.items = items
	m.selected = 0
	m.scrollOff = 0
	m.applyFilter()
	return m
}

// SetMaxHeight limits the number of visible rows. Returns a new model.
func (m SelectListModel) SetMaxHeight(h int) SelectListModel {
	m.maxHeight = h
	m.adjustScroll()
	return m
}

// SelectedItem returns the currently selected item.
// Returns a zero-value ListItem if the list is empty.
func (m SelectListModel) SelectedItem() ListItem {
	if len(m.visible) == 0 {
		return ListItem{}
	}
	return m.visible[m.selected]
}

// SelectedIndex returns the index within the visible (filtered) items.
func (m SelectListModel) SelectedIndex() int {
	return m.selected
}

// VisibleItems returns the currently filtered/visible items.
func (m SelectListModel) VisibleItems() []ListItem {
	return m.visible
}

func (m *SelectListModel) moveUp() {
	if m.selected > 0 {
		m.selected--
		m.adjustScroll()
	}
}

func (m *SelectListModel) moveDown() {
	if m.selected < len(m.visible)-1 {
		m.selected++
		m.adjustScroll()
	}
}

func (m *SelectListModel) adjustScroll() {
	if m.selected < m.scrollOff {
		m.scrollOff = m.selected
	}
	if m.selected >= m.scrollOff+m.maxHeight {
		m.scrollOff = m.selected - m.maxHeight + 1
	}
}

func (m *SelectListModel) applyFilter() {
	if m.filter == "" {
		m.visible = make([]ListItem, len(m.items))
		copy(m.visible, m.items)
		return
	}

	labels := make([]string, len(m.items))
	for i, item := range m.items {
		labels[i] = item.Label
	}
	matches := fuzzy.Find(m.filter, labels)
	m.visible = make([]ListItem, len(matches))
	for i, match := range matches {
		m.visible[i] = m.items[match.Index]
	}
}

func formatListItem(s ThemeStyles, item ListItem, w int, selected bool) string {
	var line string
	if item.Description != "" {
		line = fmt.Sprintf("  %s  %s", item.Label, item.Description)
	} else {
		line = fmt.Sprintf("  %s", item.Label)
	}

	if w > 0 {
		line = width.TruncateToWidth(line, w)
	}

	if selected {
		line = s.Bold.Render(s.Selection.Render(line))
	}
	return line
}
