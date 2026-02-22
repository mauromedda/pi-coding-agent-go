// ABOUTME: Tests for hook manager component
// ABOUTME: Validates hook CRUD, navigation, toggle, and HandleKey integration with key.Key

package components

import (
	"sort"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

func TestHookManager_New(t *testing.T) {
	hm := NewHookManager()
	if hm == nil {
		t.Fatal("NewHookManager returned nil")
	}
}

func TestHookManager_SetHooks(t *testing.T) {
	hooks := []*Hook{
		{Pattern: "test", Enabled: true},
		{Pattern: "foo", Enabled: false},
	}
	hm := NewHookManager()
	hm.SetHooks(hooks)

	if hm.Count() != 2 {
		t.Errorf("Expected 2 hooks, got %d", hm.Count())
	}
}

func TestHookManager_AddHook(t *testing.T) {
	hm := NewHookManager()
	hm.AddHook(&Hook{Pattern: "test"})

	if hm.Count() != 1 {
		t.Errorf("Expected 1 hook, got %d", hm.Count())
	}
}

func TestHookManager_RemoveHook(t *testing.T) {
	hm := NewHookManager()
	hm.AddHook(&Hook{Pattern: "a"})
	hm.AddHook(&Hook{Pattern: "b"})
	hm.AddHook(&Hook{Pattern: "c"})
	hm.selected = 1

	hm.RemoveHook()

	if hm.Count() != 2 {
		t.Errorf("Expected 2 hooks after removal, got %d", hm.Count())
	}
}

func TestHookManager_ToggleHook(t *testing.T) {
	hm := NewHookManager()
	hm.AddHook(&Hook{Pattern: "test", Enabled: false})
	hm.selected = 0

	hm.ToggleHook()

	if !hm.hooks[0].Enabled {
		t.Error("Expected hook to be enabled after toggle")
	}
}

func TestHookManager_SelectedHook(t *testing.T) {
	hm := NewHookManager()
	hm.AddHook(&Hook{Pattern: "test"})
	hm.selected = 0

	hook := hm.SelectedHook()
	if hook == nil || hook.Pattern != "test" {
		t.Error("Expected selected hook with pattern 'test'")
	}
}

func TestHookManager_RenderEmpty(t *testing.T) {
	hm := NewHookManager()

	out := &tui.RenderBuffer{}
	hm.Render(out, 80)
}

func TestHookManager_RenderWithHooks(t *testing.T) {
	hm := NewHookManager()
	hm.SetHooks([]*Hook{
		{Pattern: "test", Enabled: true},
		{Pattern: "foo", Enabled: false},
	})

	out := &tui.RenderBuffer{}
	hm.Render(out, 80)
}

func TestHookManager_Reset(t *testing.T) {
	hm := NewHookManager()
	hm.selected = 5
	hm.scrollOff = 10
	hm.SetHooks([]*Hook{{Pattern: "test"}})

	hm.Reset()

	if hm.selected != 0 {
		t.Error("Expected selected to be 0 after reset")
	}
	if hm.scrollOff != 0 {
		t.Error("Expected scrollOff to be 0 after reset")
	}
}

func TestHookManager_Count(t *testing.T) {
	hm := NewHookManager()

	if hm.Count() != 0 {
		t.Errorf("Expected 0 hooks initially, got %d", hm.Count())
	}

	hm.AddHook(&Hook{Pattern: "a"})
	hm.AddHook(&Hook{Pattern: "b"})

	if hm.Count() != 2 {
		t.Errorf("Expected 2 hooks after adding, got %d", hm.Count())
	}
}

func TestHookManager_HandleKey_Navigation(t *testing.T) {
	hm := NewHookManager()
	hm.SetHooks([]*Hook{
		{Pattern: "first", Enabled: true},
		{Pattern: "second", Enabled: false},
		{Pattern: "third", Enabled: true},
	})

	// Down
	hm.HandleKey(key.Key{Type: key.KeyDown})
	if hm.selected != 1 {
		t.Errorf("Expected selected=1 after KeyDown, got %d", hm.selected)
	}

	// Up
	hm.HandleKey(key.Key{Type: key.KeyUp})
	if hm.selected != 0 {
		t.Errorf("Expected selected=0 after KeyUp, got %d", hm.selected)
	}
}

func TestHookManager_HandleKey_EnterToggles(t *testing.T) {
	hm := NewHookManager()
	hm.SetHooks([]*Hook{
		{Pattern: "test", Enabled: false},
	})

	hm.HandleKey(key.Key{Type: key.KeyEnter})

	if !hm.hooks[0].Enabled {
		t.Error("Expected hook to be toggled on after HandleKey Enter")
	}
}

func TestConvertFromConfig(t *testing.T) {
	hooksByEvent := map[string][]config.HookDef{
		"PreToolUse": {
			{Matcher: "bash", Type: "command", Command: "echo pre-bash"},
			{Matcher: "write", Type: "command", Command: "echo pre-write"},
		},
		"PostToolUse": {
			{Matcher: "edit", Type: "command", Command: "echo post-edit"},
		},
	}

	hooks := ConvertFromConfig(hooksByEvent)
	if len(hooks) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(hooks))
	}

	// Sort by Pattern for deterministic assertion (map iteration order is random).
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Pattern < hooks[j].Pattern
	})

	tests := []struct {
		pattern string
		event   string
		enabled bool
	}{
		{"bash", "PreToolUse", true},
		{"edit", "PostToolUse", true},
		{"write", "PreToolUse", true},
	}
	for i, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if hooks[i].Pattern != tt.pattern {
				t.Errorf("hooks[%d].Pattern = %q; want %q", i, hooks[i].Pattern, tt.pattern)
			}
			if hooks[i].Event != tt.event {
				t.Errorf("hooks[%d].Event = %q; want %q", i, hooks[i].Event, tt.event)
			}
			if hooks[i].Enabled != tt.enabled {
				t.Errorf("hooks[%d].Enabled = %v; want %v", i, hooks[i].Enabled, tt.enabled)
			}
		})
	}
}

func TestHookManager_HandleKey_BoundaryDown(t *testing.T) {
	hm := NewHookManager()
	hm.SetHooks([]*Hook{
		{Pattern: "only", Enabled: true},
	})

	hm.HandleKey(key.Key{Type: key.KeyDown})
	if hm.selected != 0 {
		t.Errorf("Expected selected to stay at 0 at bottom boundary, got %d", hm.selected)
	}
}
