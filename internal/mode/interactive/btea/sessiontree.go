// ABOUTME: SessionTreeModel is a Bubble Tea overlay for browsing session trees
// ABOUTME: Supports fuzzy filter, up/down navigation, tree indent with branch indicators

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/fuzzy"
)

// SessionNode represents one node in the session tree hierarchy.
type SessionNode struct {
	ID       string
	ParentID string
	Model    string
	Count    int
	Children []*SessionNode
	Level    int
	IsBranch bool
}

// SessionTreeModel displays a tree of sessions with filter and navigation.
// Implements tea.Model with value semantics.
type SessionTreeModel struct {
	roots     []*SessionNode
	flat      []*SessionNode // flattened visible nodes
	filter    string
	selected  int
	scrollOff int
	maxHeight int
	width     int
}

// NewSessionTreeModel creates a SessionTreeModel from the given root nodes.
func NewSessionTreeModel(roots []*SessionNode) SessionTreeModel {
	m := SessionTreeModel{
		roots:     roots,
		maxHeight: 100,
	}
	m.rebuildFlat()
	return m
}

// Init returns nil; no commands needed at startup.
func (m SessionTreeModel) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation and filter clearing.
func (m SessionTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyEsc:
			if m.filter != "" {
				m.filter = ""
				m.selected = 0
				m.scrollOff = 0
				m.rebuildFlat()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the session tree with indent and branch indicators.
func (m SessionTreeModel) View() string {
	if len(m.flat) == 0 {
		return ""
	}

	s := Styles()
	end := min(m.scrollOff+m.maxHeight, len(m.flat))
	var b strings.Builder

	for i := m.scrollOff; i < end; i++ {
		node := m.flat[i]
		if i > m.scrollOff {
			b.WriteByte('\n')
		}

		line := formatTreeNode(node, m.flat, i)
		if i == m.selected {
			line = s.Bold.Render(s.Selection.Render(line))
		}
		b.WriteString(line)
	}

	return b.String()
}

// SetFilter sets the fuzzy filter string and rebuilds visible nodes. Returns a new model.
func (m SessionTreeModel) SetFilter(f string) SessionTreeModel {
	m.filter = f
	m.selected = 0
	m.scrollOff = 0
	m.rebuildFlat()
	return m
}

// SetMaxHeight limits the visible rows. Returns a new model.
func (m SessionTreeModel) SetMaxHeight(h int) SessionTreeModel {
	m.maxHeight = h
	m.adjustScroll()
	return m
}

// SelectedNode returns the currently selected node, or nil if empty.
func (m SessionTreeModel) SelectedNode() *SessionNode {
	if len(m.flat) == 0 {
		return nil
	}
	return m.flat[m.selected]
}

// Reset clears filter and selection. Returns a new model.
func (m SessionTreeModel) Reset() SessionTreeModel {
	m.filter = ""
	m.selected = 0
	m.scrollOff = 0
	m.rebuildFlat()
	return m
}

// Count returns the number of visible (flat) nodes.
func (m SessionTreeModel) Count() int {
	return len(m.flat)
}

// --- Internal helpers ---

func (m *SessionTreeModel) moveUp() {
	if m.selected > 0 {
		m.selected--
		m.adjustScroll()
	}
}

func (m *SessionTreeModel) moveDown() {
	if m.selected < len(m.flat)-1 {
		m.selected++
		m.adjustScroll()
	}
}

func (m *SessionTreeModel) adjustScroll() {
	if m.selected < m.scrollOff {
		m.scrollOff = m.selected
	}
	if m.selected >= m.scrollOff+m.maxHeight {
		m.scrollOff = m.selected - m.maxHeight + 1
	}
}

func (m *SessionTreeModel) rebuildFlat() {
	m.flat = m.flat[:0]
	for _, root := range m.roots {
		m.flattenNode(root)
	}

	if m.filter != "" {
		m.applyFilter()
	}
}

func (m *SessionTreeModel) flattenNode(node *SessionNode) {
	m.flat = append(m.flat, node)
	for _, child := range node.Children {
		m.flattenNode(child)
	}
}

func (m *SessionTreeModel) applyFilter() {
	labels := make([]string, len(m.flat))
	for i, n := range m.flat {
		labels[i] = n.ID + " " + n.Model
	}

	matches := fuzzy.Find(m.filter, labels)
	filtered := make([]*SessionNode, len(matches))
	for i, match := range matches {
		filtered[i] = m.flat[match.Index]
	}
	m.flat = filtered
}

// formatTreeNode builds a display line with tree indent and branch indicators.
func formatTreeNode(node *SessionNode, flat []*SessionNode, idx int) string {
	var prefix string
	if node.Level > 0 {
		indent := strings.Repeat("    ", node.Level-1)
		// Determine if this is the last child at its level
		isLast := isLastSibling(flat, idx)
		if isLast {
			prefix = indent + "└── "
		} else {
			prefix = indent + "├── "
		}
	}

	countStr := ""
	if node.Count > 0 {
		countStr = fmt.Sprintf(" (%d)", node.Count)
	}

	return fmt.Sprintf("%s%s [%s]%s", prefix, node.ID, node.Model, countStr)
}

// isLastSibling checks if the node at idx is the last sibling at its level
// by looking ahead in the flat list.
func isLastSibling(flat []*SessionNode, idx int) bool {
	if idx >= len(flat) {
		return true
	}
	level := flat[idx].Level
	for i := idx + 1; i < len(flat); i++ {
		if flat[i].Level <= level {
			return flat[i].Level < level
		}
	}
	return true
}
