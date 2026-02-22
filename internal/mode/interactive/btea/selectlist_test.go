// ABOUTME: Tests for SelectListModel Bubble Tea leaf component
// ABOUTME: Verifies filtering, navigation, selection, and View rendering

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: SelectListModel must satisfy tea.Model.
var _ tea.Model = SelectListModel{}

func TestSelectListModel_NewPopulatesVisible(t *testing.T) {
	items := []ListItem{
		{Label: "alpha", Description: "first"},
		{Label: "beta", Description: "second"},
		{Label: "gamma", Description: "third"},
	}
	m := NewSelectListModel(items)
	vis := m.VisibleItems()
	if len(vis) != 3 {
		t.Fatalf("VisibleItems() len = %d; want 3", len(vis))
	}
	if vis[0].Label != "alpha" {
		t.Errorf("VisibleItems()[0].Label = %q; want %q", vis[0].Label, "alpha")
	}
}

func TestSelectListModel_Init(t *testing.T) {
	m := NewSelectListModel(nil)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestSelectListModel_SetFilterReducesVisible(t *testing.T) {
	items := []ListItem{
		{Label: "apple"},
		{Label: "banana"},
		{Label: "avocado"},
	}
	m := NewSelectListModel(items)
	m = m.SetFilter("a")
	vis := m.VisibleItems()
	// "a" should fuzzy-match "apple" and "avocado" (and possibly "banana" with lower score)
	// At minimum, apple and avocado should appear
	found := 0
	for _, v := range vis {
		if v.Label == "apple" || v.Label == "avocado" {
			found++
		}
	}
	if found < 2 {
		t.Errorf("SetFilter('a') should match at least apple and avocado; got %d matches from %v", found, vis)
	}
}

func TestSelectListModel_DownUpNavigation(t *testing.T) {
	items := []ListItem{
		{Label: "one"},
		{Label: "two"},
		{Label: "three"},
	}
	m := NewSelectListModel(items)

	if m.SelectedIndex() != 0 {
		t.Fatalf("initial SelectedIndex() = %d; want 0", m.SelectedIndex())
	}

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectListModel)
	if m.SelectedIndex() != 1 {
		t.Errorf("after down: SelectedIndex() = %d; want 1", m.SelectedIndex())
	}

	// Move down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectListModel)
	if m.SelectedIndex() != 2 {
		t.Errorf("after 2x down: SelectedIndex() = %d; want 2", m.SelectedIndex())
	}

	// Move down at bottom: should stay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectListModel)
	if m.SelectedIndex() != 2 {
		t.Errorf("after down at bottom: SelectedIndex() = %d; want 2", m.SelectedIndex())
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SelectListModel)
	if m.SelectedIndex() != 1 {
		t.Errorf("after up: SelectedIndex() = %d; want 1", m.SelectedIndex())
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SelectListModel)
	if m.SelectedIndex() != 0 {
		t.Errorf("after 2x up: SelectedIndex() = %d; want 0", m.SelectedIndex())
	}

	// Move up at top: should stay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(SelectListModel)
	if m.SelectedIndex() != 0 {
		t.Errorf("after up at top: SelectedIndex() = %d; want 0", m.SelectedIndex())
	}
}

func TestSelectListModel_SelectedItem(t *testing.T) {
	items := []ListItem{
		{Label: "one", Description: "first"},
		{Label: "two", Description: "second"},
	}
	m := NewSelectListModel(items)
	sel := m.SelectedItem()
	if sel.Label != "one" {
		t.Errorf("SelectedItem().Label = %q; want %q", sel.Label, "one")
	}

	// Move down and check
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SelectListModel)
	sel = m.SelectedItem()
	if sel.Label != "two" {
		t.Errorf("after down: SelectedItem().Label = %q; want %q", sel.Label, "two")
	}
}

func TestSelectListModel_SelectedItemEmpty(t *testing.T) {
	m := NewSelectListModel(nil)
	sel := m.SelectedItem()
	if sel.Label != "" {
		t.Errorf("SelectedItem() on empty list = %+v; want zero value", sel)
	}
}

func TestSelectListModel_ViewContainsSelectedLabel(t *testing.T) {
	items := []ListItem{
		{Label: "alpha", Description: "letter a"},
		{Label: "beta", Description: "letter b"},
	}
	m := NewSelectListModel(items)
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "alpha") {
		t.Errorf("View() missing selected item label 'alpha'")
	}
	if !strings.Contains(view, "beta") {
		t.Errorf("View() missing item label 'beta'")
	}
}

func TestSelectListModel_ViewEmptyList(t *testing.T) {
	m := NewSelectListModel(nil)
	view := m.View()
	if view != "" {
		t.Errorf("View() on empty list = %q; want empty", view)
	}
}

func TestSelectListModel_WindowSizeMsg(t *testing.T) {
	m := NewSelectListModel(nil)
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	w := updated.(SelectListModel)
	if w.width != 120 {
		t.Errorf("width = %d; want 120", w.width)
	}
}

func TestSelectListModel_SetItems(t *testing.T) {
	m := NewSelectListModel([]ListItem{{Label: "old"}})
	newItems := []ListItem{
		{Label: "new1"},
		{Label: "new2"},
	}
	m = m.SetItems(newItems)
	vis := m.VisibleItems()
	if len(vis) != 2 {
		t.Fatalf("after SetItems: VisibleItems() len = %d; want 2", len(vis))
	}
	if vis[0].Label != "new1" {
		t.Errorf("after SetItems: VisibleItems()[0].Label = %q; want %q", vis[0].Label, "new1")
	}
}

func TestSelectListModel_SetMaxHeight(t *testing.T) {
	items := make([]ListItem, 20)
	for i := range items {
		items[i] = ListItem{Label: "item"}
	}
	m := NewSelectListModel(items)
	m = m.SetMaxHeight(5)
	m.width = 80

	view := m.View()
	lines := strings.Split(view, "\n")
	// Remove trailing empty line if present
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 5 {
		t.Errorf("View() with maxHeight=5 rendered %d lines; want <= 5", len(lines))
	}
}

func TestSelectListModel_ScrollingOnNavigation(t *testing.T) {
	items := make([]ListItem, 20)
	for i := range items {
		items[i] = ListItem{Label: "item"}
	}
	m := NewSelectListModel(items)
	m = m.SetMaxHeight(3)

	// Navigate down past visible window
	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(SelectListModel)
	}
	if m.SelectedIndex() != 5 {
		t.Errorf("after 5x down: SelectedIndex() = %d; want 5", m.SelectedIndex())
	}
	// scrollOff should have adjusted so selected is visible
	if m.scrollOff > m.selected {
		t.Errorf("scrollOff=%d > selected=%d; selected should be visible", m.scrollOff, m.selected)
	}
	if m.selected >= m.scrollOff+m.maxHeight {
		t.Errorf("selected=%d >= scrollOff(%d)+maxHeight(%d); selected should be visible",
			m.selected, m.scrollOff, m.maxHeight)
	}
}
