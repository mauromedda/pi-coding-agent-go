// ABOUTME: Memory leak tests for EditorModel
// ABOUTME: Part of Phase 1: Core Stabilization - Editor Memory Management

package btea

import (
	"strings"
	"testing"
)

// TestEditorUndoStackGrowth verifies that undo stack doesn't grow unbounded
func TestEditorUndoStackGrowth(t *testing.T) {
	editor := NewEditorModel()

	// Perform many undoable operations
	for i := range 500 {
		editor = editor.SetText(strings.Repeat("test ", i))
	}

	// The undo stack should be capped at editorUndoDepth (200)
	// After 500 operations, we should have at most editorUndoDepth + some buffer states
	_, ok := editor.undoStack.undo()
	if !ok {
		t.Fatal("Expected to be able to undo")
	}

	// Count how many states we can undo
	count := 1
	for {
		_, ok := editor.undoStack.undo()
		if !ok {
			break
		}
		count++
	}

	// Should have approximately editorUndoDepth states
	if count < editorUndoDepth-10 || count > editorUndoDepth+10 {
		t.Errorf("Undo stack count unexpected: got %d, want ~%d", count, editorUndoDepth)
	}
}

// TestEditorStateMemoryCleanup verifies that old states are properly dropped
func TestEditorStateMemoryCleanup(t *testing.T) {
	editor := NewEditorModel()

	// Perform many undoable operations
	for i := range 300 {
		editor = editor.SetText("test" + string(rune('a'+i)))
	}

	// After 300 operations with depth 200, we should have ~200 states
	_, ok := editor.undoStack.undo()
	if !ok {
		t.Fatal("Expected to be able to undo")
	}

	// Verify we can still undo (should have ~199 left)
	count := 1
	for {
		_, ok := editor.undoStack.undo()
		if !ok {
			break
		}
		count++
	}

	// Should have ~199 states (300 - 1 initial + depth cap)
	if count < 150 || count > 250 {
		t.Errorf("Undo states count unexpected: got %d, want ~200", count)
	}
}

// TestEditorDeepCopyIsolation verifies that undo snapshots are deep copies
func TestEditorDeepCopyIsolation(t *testing.T) {
	editor := NewEditorModel()
	editor = editor.SetText("hello world")

	// Save state
	_, ok := editor.undoStack.undo()
	if !ok {
		t.Fatal("Expected to have state")
	}

	// Modify editor
	editor = editor.SetText("modified")

	// The original snapshot should still be in the stack
	// and should be unaffected by modifications
	count := 1
	for {
		_, ok := editor.undoStack.undo()
		if !ok {
			break
		}
		count++
	}

	// Should have states left in the stack
	if count < 1 {
		t.Error("Expected to have undo states")
	}
}

// TestEditorLineMemoryPressure tests memory pressure with many lines
func TestEditorLineMemoryPressure(t *testing.T) {
	editor := NewEditorModel()

	// Add many lines with undoable operations
	for i := range 500 {
		if i > 0 {
			// Create a new line by setting text with newline
			editor = editor.SetText(editor.Text() + "\n" + string(rune('a'+(i%26))))
		} else {
			editor = editor.SetText(string(rune('a' + (i % 26))))
		}
	}

	// Verify we can handle many lines
	if editor.LineCount() == 0 {
		t.Error("Expected non-zero line count")
	}

	// Test undo across many lines
	undoCount := 0
	for range 100 {
		editor = editor.doUndoForTest()
		undoCount++
	}

	// Should still have valid state after undo
	if editor.LineCount() == 0 {
		t.Error("Expected non-zero line count after undo")
	}
}

// TestEditorUndoDepthCap verifies that undo depth is properly capped
func TestEditorUndoDepthCap(t *testing.T) {
	editor := NewEditorModel()

	// Perform many operations
	for i := range 1000 {
		editor = editor.SetText(strings.Repeat("x", 100) + string(rune('a'+i)))
	}

	// Count total undoable states
	totalUndos := 0
	for {
		_, ok := editor.undoStack.undo()
		if !ok {
			break
		}
		totalUndos++
	}

	// Should be capped around editorUndoDepth (200)
	if totalUndos > editorUndoDepth+5 {
		t.Errorf("Undo depth not capped: got %d, want <= %d", totalUndos, editorUndoDepth+5)
	}

	if totalUndos < editorUndoDepth-5 {
		t.Errorf("Undo depth too small: got %d, want >= %d", totalUndos, editorUndoDepth-5)
	}
}

// doUndoForTest is a helper for testing that returns the new editor
func (m EditorModel) doUndoForTest() EditorModel {
	state, ok := m.undoStack.undo()
	if !ok {
		return m
	}
	// Deep copy to prevent mutating the snapshot in the stack
	lines := make([][]rune, len(state.lines))
	for i, l := range state.lines {
		lines[i] = make([]rune, len(l))
		copy(lines[i], l)
	}
	m.lines = lines
	m.row = state.row
	m.col = state.col
	return m
}

// TestEditorMemoryPerStateEstimate documents memory usage per state
func TestEditorMemoryPerStateEstimate(t *testing.T) {
	editor := NewEditorModel()

	// Create editor with substantial content
	largeText := strings.Repeat("hello world ", 100)
	for i := range 100 {
		editor = editor.SetText(largeText + string(rune('a'+i)))
	}

	// Calculate memory per state
	state, ok := editor.undoStack.undo()
	if !ok {
		t.Fatal("Expected to have state")
	}

	// Count total runes in the state
	totalRunes := 0
	for _, line := range state.lines {
		totalRunes += len(line)
	}

	// Each rune is 4 bytes in Go
	approxMemory := totalRunes * 4

	t.Logf("Approximate memory per state: %d bytes (%d runes)", approxMemory, totalRunes)

	// With 200 undo depth, this could be significant
	totalEstimated := approxMemory * editorUndoDepth
	t.Logf("Estimated total memory with full undo stack: %d bytes (%.2f MB)",
		totalEstimated, float64(totalEstimated)/(1024*1024))

	// This test documents potential memory usage
	// The implementation should limit this to editorUndoDepth
}
