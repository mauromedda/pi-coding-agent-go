// ABOUTME: Tests for FileMentionSelector component

package component

import (
	"fmt"
	"testing"
)

func TestFileMentionSelector_Create(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")
	if fm == nil {
		t.Fatal("NewFileMentionSelector returned nil")
	}
}

func TestFileMentionSelector_SetFilter(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")

	// Set filter should trigger filtering
	fm.SetFilter("main.go")

	if fm.filter != "main.go" {
		t.Errorf("Expected filter to be 'main.go', got '%s'", fm.filter)
	}
}

func TestFileMentionSelector_MoveUp(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")

	// Add some fake items
	fm.items = []FileInfo{
		{RelPath: "file1.go"},
		{RelPath: "file2.go"},
		{RelPath: "file3.go"},
	}
	fm.visible = fm.items
	fm.selected = 1
	fm.scrollOff = 0

	// Move up
	fm.moveUp()

	if fm.selected != 0 {
		t.Errorf("Expected selected to be 0 after moveUp, got %d", fm.selected)
	}
}

func TestFileMentionSelector_MoveDown(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")

	// Add some fake items
	fm.items = []FileInfo{
		{RelPath: "file1.go"},
		{RelPath: "file2.go"},
		{RelPath: "file3.go"},
	}
	fm.visible = fm.items
	fm.selected = 0
	fm.scrollOff = 0

	// Move down
	fm.moveDown()

	if fm.selected != 1 {
		t.Errorf("Expected selected to be 1 after moveDown, got %d", fm.selected)
	}
}

func TestFileMentionSelector_SelectItem(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")

	// Add some fake items
	fm.items = []FileInfo{
		{RelPath: "src/main.go"},
		{RelPath: "src/utils.go"},
		{RelPath: "README.md"},
	}
	fm.visible = fm.items
	fm.selected = 0

	item := fm.SelectedItem()

	if item.RelPath != "src/main.go" {
		t.Errorf("Expected first item to be 'src/main.go', got '%s'", item.RelPath)
	}
}

func TestFileMentionSelector_ApplyFilter(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")

	// Add some fake items
	fm.items = []FileInfo{
		{RelPath: "src/main.go"},
		{RelPath: "src/utils.go"},
		{RelPath: "README.md"},
		{RelPath: "test/test.go"},
	}

	// Apply filter for "main"
	fm.SetFilter("main")

	if len(fm.visible) == 0 {
		t.Error("Expected at least one match for 'main'")
	}
}

func TestFileMentionSelector_ScrollAdjustment(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")
	fm.maxHeight = 5

	// Add more items than max height
	for i := range 20 {
		fm.items = append(fm.items, FileInfo{RelPath: fmt.Sprintf("file%d.go", i)})
	}
	fm.visible = fm.items

	// Set selection near the end
	fm.selected = 15
	fm.scrollOff = 0

	fm.adjustScroll()

	// Should have scrolled down
	if fm.scrollOff == 0 {
		t.Error("Expected scrollOff to be non-zero when selection is at end")
	}
}

func TestFileMentionSelector_SelectionAccepted_NoDeadlock(t *testing.T) {
	t.Parallel()

	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")
	fm.items = []FileInfo{
		{Path: "/tmp/test/main.go", RelPath: "main.go"},
		{Path: "/tmp/test/util.go", RelPath: "util.go"},
	}
	fm.visible = fm.items
	fm.selected = 0

	// SelectionAccepted must not deadlock; it calls SelectedItem() which
	// also locks the mutex. A timeout-based approach is fragile, so we
	// just call it and verify the result — if it deadlocks, the test
	// runner will kill us after the test timeout.
	got := fm.SelectionAccepted()
	if got != "main.go" {
		t.Errorf("SelectionAccepted() = %q, want %q", got, "main.go")
	}
}

func TestFileMentionSelector_HandleInput_UpNoDeadlock(t *testing.T) {
	t.Parallel()

	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")
	fm.items = []FileInfo{
		{RelPath: "a.go"},
		{RelPath: "b.go"},
		{RelPath: "c.go"},
	}
	fm.visible = fm.items
	fm.selected = 2

	// HandleInput with Up arrow escape sequence calls moveUpLocked()
	// which calls adjustScroll() that re-locks the mutex. This
	// deadlocks if not fixed.
	fm.HandleInput("\x1b[A") // Up arrow escape sequence

	if fm.selected != 1 {
		t.Errorf("selected = %d after Up, want 1", fm.selected)
	}
}

func TestFileMentionSelector_Reset(t *testing.T) {
	fm := NewFileMentionSelector("/tmp/test", "/tmp/test")

	fm.filter = "test"
	fm.selected = 5
	fm.scrollOff = 10

	fm.Reset()

	if fm.filter != "" {
		t.Error("Expected filter to be empty after Reset")
	}
	if fm.selected != 0 {
		t.Error("Expected selected to be 0 after Reset")
	}
	if fm.scrollOff != 0 {
		t.Error("Expected scrollOff to be 0 after Reset")
	}
}
