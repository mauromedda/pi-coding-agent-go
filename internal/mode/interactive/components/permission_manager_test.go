// ABOUTME: Tests for permission manager component
// ABOUTME: Validates rule CRUD, navigation, and HandleKey integration with key.Key

package components

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

func TestPermissionManager_New(t *testing.T) {
	pm := NewPermissionManager()
	if pm == nil {
		t.Fatal("NewPermissionManager returned nil")
	}
}

func TestPermissionManager_SetRules(t *testing.T) {
	rules := []*permission.Rule{
		{Tool: "*.go", Allow: true},
		{Tool: "*.env", Allow: false},
	}
	pm := NewPermissionManager()
	pm.SetRules(rules)

	if pm.Count() != 2 {
		t.Errorf("Expected 2 rules, got %d", pm.Count())
	}
}

func TestPermissionManager_AddRule(t *testing.T) {
	pm := NewPermissionManager()
	pm.AddRule(&permission.Rule{Tool: "*.test", Allow: true})

	if pm.Count() != 1 {
		t.Errorf("Expected 1 rule, got %d", pm.Count())
	}
}

func TestPermissionManager_RemoveRule(t *testing.T) {
	pm := NewPermissionManager()
	pm.AddRule(&permission.Rule{Tool: "a"})
	pm.AddRule(&permission.Rule{Tool: "b"})
	pm.AddRule(&permission.Rule{Tool: "c"})
	pm.selected = 1

	pm.RemoveRule()

	if pm.Count() != 2 {
		t.Errorf("Expected 2 rules after removal, got %d", pm.Count())
	}
}

func TestPermissionManager_SelectedRule(t *testing.T) {
	pm := NewPermissionManager()
	pm.AddRule(&permission.Rule{Tool: "test", Allow: true})
	pm.selected = 0

	rule := pm.SelectedRule()
	if rule == nil || rule.Tool != "test" {
		t.Error("Expected selected rule with tool 'test'")
	}
}

func TestPermissionManager_RenderEmpty(t *testing.T) {
	pm := NewPermissionManager()

	out := &tui.RenderBuffer{}
	pm.Render(out, 80)
}

func TestPermissionManager_RenderWithRules(t *testing.T) {
	pm := NewPermissionManager()
	pm.SetRules([]*permission.Rule{
		{Tool: "*.go", Allow: true},
		{Tool: "*.env", Allow: false},
	})

	out := &tui.RenderBuffer{}
	pm.Render(out, 80)
}

func TestPermissionManager_Reset(t *testing.T) {
	pm := NewPermissionManager()
	pm.selected = 5
	pm.scrollOff = 10
	pm.SetRules([]*permission.Rule{{Tool: "test", Allow: true}})

	pm.Reset()

	if pm.selected != 0 {
		t.Error("Expected selected to be 0 after reset")
	}
	if pm.scrollOff != 0 {
		t.Error("Expected scrollOff to be 0 after reset")
	}
}

func TestPermissionManager_Count(t *testing.T) {
	pm := NewPermissionManager()

	if pm.Count() != 0 {
		t.Errorf("Expected 0 rules initially, got %d", pm.Count())
	}

	pm.AddRule(&permission.Rule{Tool: "a", Allow: true})
	pm.AddRule(&permission.Rule{Tool: "b", Allow: false})

	if pm.Count() != 2 {
		t.Errorf("Expected 2 rules after adding, got %d", pm.Count())
	}
}

func TestPermissionManager_HandleKey_Navigation(t *testing.T) {
	pm := NewPermissionManager()
	pm.SetRules([]*permission.Rule{
		{Tool: "read", Allow: true},
		{Tool: "write", Allow: true},
		{Tool: "bash", Allow: false},
	})

	// Down
	pm.HandleKey(key.Key{Type: key.KeyDown})
	if pm.selected != 1 {
		t.Errorf("Expected selected=1 after KeyDown, got %d", pm.selected)
	}

	// Down again
	pm.HandleKey(key.Key{Type: key.KeyDown})
	if pm.selected != 2 {
		t.Errorf("Expected selected=2 after second KeyDown, got %d", pm.selected)
	}

	// Up
	pm.HandleKey(key.Key{Type: key.KeyUp})
	if pm.selected != 1 {
		t.Errorf("Expected selected=1 after KeyUp, got %d", pm.selected)
	}
}

func TestPermissionManager_HandleKey_BoundaryUp(t *testing.T) {
	pm := NewPermissionManager()
	pm.SetRules([]*permission.Rule{
		{Tool: "read", Allow: true},
	})

	pm.HandleKey(key.Key{Type: key.KeyUp})
	if pm.selected != 0 {
		t.Errorf("Expected selected to stay at 0 at top boundary, got %d", pm.selected)
	}
}
