// ABOUTME: Tests for session tree component
// ABOUTME: Validates navigation, filtering, and HandleKey integration with key.Key

package components

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

func TestSessionTree_Create(t *testing.T) {
	st := NewSessionTree(nil)
	if st == nil {
		t.Fatal("NewSessionTree returned nil")
	}
}

func TestSessionTree_SetFilter(t *testing.T) {
	st := NewSessionTree(nil)
	st.SetFilter("hello")

	if st.filter != "hello" {
		t.Errorf("Expected filter 'hello', got '%s'", st.filter)
	}
}

func TestSessionTree_MoveUp(t *testing.T) {
	st := NewSessionTree(nil)
	st.selected = 1

	st.moveUp()

	if st.selected != 0 {
		t.Errorf("Expected selected to be 0 after moveUp, got %d", st.selected)
	}
}

func TestSessionTree_MoveDown(t *testing.T) {
	st := NewSessionTree(nil)
	st.selected = 0

	st.moveDown()

	if st.selected != 0 {
		// Nothing to move down to
		t.Errorf("Expected selected to stay at 0, got %d", st.selected)
	}
}

func TestSessionTree_SelectedNode(t *testing.T) {
	st := NewSessionTree(nil)
	st.selected = 0

	node := st.SelectedNode()
	if node != nil {
		t.Error("Expected nil node for empty tree")
	}
}

func TestSessionTree_Count(t *testing.T) {
	st := NewSessionTree(nil)

	count := st.Count()
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestSessionTree_RenderEmpty(t *testing.T) {
	st := NewSessionTree(nil)

	out := &tui.RenderBuffer{}
	st.Render(out, 80)
}

func TestSessionTree_Reset(t *testing.T) {
	st := NewSessionTree(nil)
	st.filter = "test"
	st.selected = 5
	st.scrollOff = 10

	st.Reset()

	if st.filter != "" {
		t.Error("Expected filter to be empty after Reset")
	}
	if st.selected != 0 {
		t.Error("Expected selected to be 0 after Reset")
	}
}

func TestSessionTree_HandleKey_Navigation(t *testing.T) {
	nodes := []*SessionNode{
		{ID: "aaaa1111-0000-0000-0000-000000000001", Model: "gpt-4"},
		{ID: "aaaa1111-0000-0000-0000-000000000002", Model: "claude"},
		{ID: "aaaa1111-0000-0000-0000-000000000003", Model: "gemini"},
	}
	st := NewSessionTree(nodes)

	// Down
	st.HandleKey(key.Key{Type: key.KeyDown})
	if st.selected != 1 {
		t.Errorf("Expected selected=1 after KeyDown, got %d", st.selected)
	}

	// Down again
	st.HandleKey(key.Key{Type: key.KeyDown})
	if st.selected != 2 {
		t.Errorf("Expected selected=2 after second KeyDown, got %d", st.selected)
	}

	// Up
	st.HandleKey(key.Key{Type: key.KeyUp})
	if st.selected != 1 {
		t.Errorf("Expected selected=1 after KeyUp, got %d", st.selected)
	}
}

func TestSessionTree_HandleKey_Escape(t *testing.T) {
	nodes := []*SessionNode{
		{ID: "aaaa1111-0000-0000-0000-000000000001", Model: "gpt-4"},
	}
	st := NewSessionTree(nodes)
	st.SetFilter("test")
	st.selected = 0

	st.HandleKey(key.Key{Type: key.KeyEscape})

	if st.filter != "" {
		t.Errorf("Expected filter cleared after Escape, got %q", st.filter)
	}
	if st.selected != 0 {
		t.Errorf("Expected selected=0 after Escape, got %d", st.selected)
	}
}

func TestSessionTree_HandleKey_BoundaryUp(t *testing.T) {
	nodes := []*SessionNode{
		{ID: "aaaa1111-0000-0000-0000-000000000001", Model: "gpt-4"},
	}
	st := NewSessionTree(nodes)
	st.selected = 0

	st.HandleKey(key.Key{Type: key.KeyUp})
	if st.selected != 0 {
		t.Errorf("Expected selected to stay at 0 at top boundary, got %d", st.selected)
	}
}
