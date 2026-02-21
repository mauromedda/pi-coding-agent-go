// ABOUTME: Tests for tool registry: registration, lookup, and metadata validation
// ABOUTME: Verifies all built-in tools are present with correct attributes

package tools

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

func TestNewRegistry_RegistersAllBuiltins(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	all := r.All()

	expectedTools := []string{"read", "write", "edit", "bash", "grep", "find", "ls"}
	if len(all) < len(expectedTools) {
		t.Errorf("expected at least %d tools, got %d", len(expectedTools), len(all))
	}

	for _, name := range expectedTools {
		if r.Get(name) == nil {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
}

func TestRegistry_Get_ReturnsNilForUnknown(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if got := r.Get("nonexistent"); got != nil {
		t.Errorf("expected nil for unknown tool, got %v", got)
	}
}

func TestRegistry_ReadOnly_FiltersCorrectly(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	roTools := r.ReadOnly()

	for _, tool := range roTools {
		if !tool.ReadOnly {
			t.Errorf("ReadOnly() returned non-read-only tool %q", tool.Name)
		}
	}

	// read, grep, find, ls should be read-only
	expectedReadOnly := map[string]bool{"read": true, "grep": true, "find": true, "ls": true}
	for _, tool := range roTools {
		if !expectedReadOnly[tool.Name] {
			t.Errorf("unexpected read-only tool: %q", tool.Name)
		}
	}
	if len(roTools) < len(expectedReadOnly) {
		t.Errorf("expected at least %d read-only tools, got %d", len(expectedReadOnly), len(roTools))
	}
}

func TestRegistry_Register_OverridesExisting(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	custom := &agent.AgentTool{
		Name:        "read",
		Label:       "Custom Read",
		Description: "overridden",
	}
	r.Register(custom)

	got := r.Get("read")
	if got.Description != "overridden" {
		t.Errorf("expected overridden description, got %q", got.Description)
	}
}

func TestRegistry_ToolMetadata(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	tests := []struct {
		name     string
		readOnly bool
	}{
		{"read", true},
		{"write", false},
		{"edit", false},
		{"bash", false},
		{"grep", true},
		{"find", true},
		{"ls", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tool := r.Get(tt.name)
			if tool == nil {
				t.Fatalf("tool %q not found", tt.name)
			}
			if tool.ReadOnly != tt.readOnly {
				t.Errorf("tool %q: ReadOnly = %v, want %v", tt.name, tool.ReadOnly, tt.readOnly)
			}
			if tool.Description == "" {
				t.Errorf("tool %q has empty description", tt.name)
			}
			if tool.Parameters == nil {
				t.Errorf("tool %q has nil parameters", tt.name)
			}
			if tool.Execute == nil {
				t.Errorf("tool %q has nil Execute function", tt.name)
			}
		})
	}
}
