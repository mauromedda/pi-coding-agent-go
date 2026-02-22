// ABOUTME: Tests for SessionTreeModel overlay: tree rendering, navigation, filter
// ABOUTME: Verifies indent levels, branch indicators, fuzzy filter, and selection

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: SessionTreeModel must satisfy tea.Model.
var _ tea.Model = SessionTreeModel{}

func buildTestTree() []*SessionNode {
	child1 := &SessionNode{
		ID:    "child-1",
		Model: "opus",
		Count: 3,
		Level: 1,
	}
	child2 := &SessionNode{
		ID:    "child-2",
		Model: "sonnet",
		Count: 1,
		Level: 1,
	}
	root := &SessionNode{
		ID:       "root-1",
		Model:    "opus",
		Count:    10,
		Level:    0,
		IsBranch: true,
		Children: []*SessionNode{child1, child2},
	}
	child1.ParentID = "root-1"
	child2.ParentID = "root-1"
	return []*SessionNode{root}
}

func TestSessionTreeModel_Init(t *testing.T) {
	m := NewSessionTreeModel(nil)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestSessionTreeModel_Count(t *testing.T) {
	tests := []struct {
		name  string
		roots []*SessionNode
		want  int
	}{
		{"nil roots", nil, 0},
		{"single root no children", []*SessionNode{{ID: "r1", Model: "opus"}}, 1},
		{"root with children", buildTestTree(), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSessionTreeModel(tt.roots)
			if got := m.Count(); got != tt.want {
				t.Errorf("Count() = %d; want %d", got, tt.want)
			}
		})
	}
}

func TestSessionTreeModel_SelectedNode(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)

	sel := m.SelectedNode()
	if sel == nil {
		t.Fatal("SelectedNode() returned nil; want root node")
	}
	if sel.ID != "root-1" {
		t.Errorf("SelectedNode().ID = %q; want %q", sel.ID, "root-1")
	}
}

func TestSessionTreeModel_SelectedNodeEmpty(t *testing.T) {
	m := NewSessionTreeModel(nil)
	if sel := m.SelectedNode(); sel != nil {
		t.Errorf("SelectedNode() on empty tree = %+v; want nil", sel)
	}
}

func TestSessionTreeModel_Navigation(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)

	// Move down: root-1 -> child-1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionTreeModel)
	sel := m.SelectedNode()
	if sel == nil || sel.ID != "child-1" {
		t.Fatalf("after down: SelectedNode().ID = %v; want child-1", sel)
	}

	// Move down: child-1 -> child-2
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionTreeModel)
	sel = m.SelectedNode()
	if sel == nil || sel.ID != "child-2" {
		t.Fatalf("after 2x down: SelectedNode().ID = %v; want child-2", sel)
	}

	// Move down at bottom: stays at child-2
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionTreeModel)
	sel = m.SelectedNode()
	if sel == nil || sel.ID != "child-2" {
		t.Fatalf("after down at bottom: SelectedNode().ID = %v; want child-2", sel)
	}

	// Move up: child-2 -> child-1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SessionTreeModel)
	sel = m.SelectedNode()
	if sel == nil || sel.ID != "child-1" {
		t.Fatalf("after up: SelectedNode().ID = %v; want child-1", sel)
	}

	// Move up: child-1 -> root-1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SessionTreeModel)
	sel = m.SelectedNode()
	if sel == nil || sel.ID != "root-1" {
		t.Fatalf("after 2x up: SelectedNode().ID = %v; want root-1", sel)
	}
}

func TestSessionTreeModel_ViewRendersTree(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)
	m.width = 80
	m.maxHeight = 20

	view := m.View()
	if !strings.Contains(view, "root-1") {
		t.Errorf("View() missing root ID 'root-1'")
	}
	if !strings.Contains(view, "child-1") {
		t.Errorf("View() missing child ID 'child-1'")
	}
	if !strings.Contains(view, "opus") {
		t.Errorf("View() missing model name 'opus'")
	}
}

func TestSessionTreeModel_ViewBranchIndicators(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)
	m.width = 80
	m.maxHeight = 20

	view := m.View()
	// Children should have tree indicators
	if !strings.Contains(view, "├──") && !strings.Contains(view, "└──") {
		t.Errorf("View() missing tree branch indicators (├── or └──)")
	}
}

func TestSessionTreeModel_ViewEmpty(t *testing.T) {
	m := NewSessionTreeModel(nil)
	m.width = 80
	view := m.View()
	if view != "" {
		t.Errorf("View() on empty tree = %q; want empty", view)
	}
}

func TestSessionTreeModel_SetFilter(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)

	m = m.SetFilter("sonnet")
	// After filtering for "sonnet", only matching nodes should be visible
	if m.Count() == 0 {
		t.Error("SetFilter('sonnet') resulted in 0 count; want at least 1 match")
	}
}

func TestSessionTreeModel_SetFilterEmpty(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)
	m = m.SetFilter("sonnet")
	m = m.SetFilter("")
	// Clearing filter should restore all nodes
	if m.Count() != 3 {
		t.Errorf("after clearing filter: Count() = %d; want 3", m.Count())
	}
}

func TestSessionTreeModel_SetMaxHeight(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)
	m = m.SetMaxHeight(2)
	m.width = 80

	view := m.View()
	lines := strings.Split(view, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 2 {
		t.Errorf("View() with maxHeight=2 rendered %d lines; want <= 2", len(lines))
	}
}

func TestSessionTreeModel_Reset(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)

	// Navigate and filter
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionTreeModel)
	m = m.SetFilter("sonnet")

	// Reset should clear filter and selection
	m = m.Reset()
	if sel := m.SelectedNode(); sel == nil || sel.ID != "root-1" {
		t.Errorf("after Reset: SelectedNode() = %v; want root-1", sel)
	}
}

func TestSessionTreeModel_EscClearsFilter(t *testing.T) {
	roots := buildTestTree()
	m := NewSessionTreeModel(roots)
	m = m.SetFilter("sonnet")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(SessionTreeModel)

	if m.filter != "" {
		t.Errorf("after esc: filter = %q; want empty", m.filter)
	}
}
