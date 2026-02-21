// ABOUTME: Tests for the filterable scrollable list component
// ABOUTME: Covers navigation, fuzzy filtering, selection, viewport scrolling, rendering

package component

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func makeItems() []ListItem {
	return []ListItem{
		{Label: "Apple", Description: "A fruit"},
		{Label: "Banana", Description: "Yellow fruit"},
		{Label: "Cherry", Description: "Small red fruit"},
		{Label: "Date", Description: "Sweet fruit"},
		{Label: "Elderberry", Description: "Dark berry"},
	}
}

func TestSelectList_New(t *testing.T) {
	t.Parallel()

	items := makeItems()
	sl := NewSelectList(items)

	if sl.SelectedIndex() != 0 {
		t.Errorf("expected selected index 0, got %d", sl.SelectedIndex())
	}
	if sl.SelectedItem().Label != "Apple" {
		t.Errorf("expected 'Apple', got %q", sl.SelectedItem().Label)
	}
}

func TestSelectList_MoveDown(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.HandleInput("\x1b[B") // down

	if sl.SelectedIndex() != 1 {
		t.Errorf("expected index 1, got %d", sl.SelectedIndex())
	}
	if sl.SelectedItem().Label != "Banana" {
		t.Errorf("expected 'Banana', got %q", sl.SelectedItem().Label)
	}
}

func TestSelectList_MoveUp(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.HandleInput("\x1b[B") // down
	sl.HandleInput("\x1b[B") // down
	sl.HandleInput("\x1b[A") // up

	if sl.SelectedIndex() != 1 {
		t.Errorf("expected index 1, got %d", sl.SelectedIndex())
	}
}

func TestSelectList_MoveUpAtTop(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.HandleInput("\x1b[A") // up at top -> no change

	if sl.SelectedIndex() != 0 {
		t.Errorf("expected index 0, got %d", sl.SelectedIndex())
	}
}

func TestSelectList_MoveDownAtBottom(t *testing.T) {
	t.Parallel()

	items := makeItems()
	sl := NewSelectList(items)
	for range items {
		sl.HandleInput("\x1b[B")
	}

	if sl.SelectedIndex() != len(items)-1 {
		t.Errorf("expected index %d, got %d", len(items)-1, sl.SelectedIndex())
	}
}

func TestSelectList_Filter(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetFilter("ban")

	visible := sl.VisibleItems()
	if len(visible) == 0 {
		t.Fatal("expected at least one match for 'ban'")
	}
	if visible[0].Label != "Banana" {
		t.Errorf("expected 'Banana' as top match, got %q", visible[0].Label)
	}
}

func TestSelectList_FilterNoMatch(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetFilter("zzz")

	visible := sl.VisibleItems()
	if len(visible) != 0 {
		t.Errorf("expected 0 matches for 'zzz', got %d", len(visible))
	}
}

func TestSelectList_ClearFilter(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetFilter("ban")
	sl.SetFilter("")

	visible := sl.VisibleItems()
	if len(visible) != 5 {
		t.Errorf("expected 5 items after clearing filter, got %d", len(visible))
	}
}

func TestSelectList_RenderBasic(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetMaxHeight(10)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sl.Render(buf, 40)

	if buf.Len() < 5 {
		t.Fatalf("expected at least 5 lines, got %d", buf.Len())
	}
}

func TestSelectList_RenderHighlightsSelected(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetMaxHeight(10)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sl.Render(buf, 40)

	// First item should have highlight marker (bold or reverse)
	if !strings.Contains(buf.Lines[0], "\x1b[") {
		t.Error("expected ANSI formatting on selected item")
	}
}

func TestSelectList_RenderViewport(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetMaxHeight(3) // only 3 visible

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sl.Render(buf, 40)

	if buf.Len() > 3 {
		t.Errorf("expected at most 3 lines with maxHeight=3, got %d", buf.Len())
	}
}

func TestSelectList_ScrollViewport(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetMaxHeight(3)

	// Move down past the viewport
	sl.HandleInput("\x1b[B") // 1
	sl.HandleInput("\x1b[B") // 2
	sl.HandleInput("\x1b[B") // 3

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sl.Render(buf, 40)

	// The selected item (index 3, "Date") should be visible
	found := false
	for _, line := range buf.Lines {
		if strings.Contains(line, "Date") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Date' to be visible after scrolling")
	}
}

func TestSelectList_EmptyList(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(nil)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sl.Render(buf, 40)

	// Should not crash; may render empty
	if buf.Len() > 1 {
		t.Errorf("expected 0 or 1 lines for empty list, got %d", buf.Len())
	}
}

func TestSelectList_SetItems(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	newItems := []ListItem{
		{Label: "Xylophone"},
		{Label: "Yacht"},
	}
	sl.SetItems(newItems)

	if sl.SelectedIndex() != 0 {
		t.Errorf("expected index reset to 0, got %d", sl.SelectedIndex())
	}
	visible := sl.VisibleItems()
	if len(visible) != 2 {
		t.Errorf("expected 2 items, got %d", len(visible))
	}
}

func TestSelectList_Invalidate(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.Invalidate()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	sl.Render(buf, 40)
	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line after invalidate")
	}
}

func TestSelectList_FilterThenNavigate(t *testing.T) {
	t.Parallel()

	sl := NewSelectList(makeItems())
	sl.SetFilter("berry")
	sl.HandleInput("\x1b[B") // down; should not go past visible items

	visible := sl.VisibleItems()
	if len(visible) == 0 {
		t.Skip("no matches for 'berry'; skipping navigation test")
	}
	if sl.SelectedIndex() >= len(visible) {
		t.Errorf("selected index %d out of bounds for %d visible items", sl.SelectedIndex(), len(visible))
	}
}
