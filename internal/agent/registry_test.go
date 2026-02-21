// ABOUTME: Tests for the agent definition registry
// ABOUTME: Covers builtin loading, custom override, list, and register operations

package agent

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestRegistry_BuiltinsLoaded(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(t.TempDir(), t.TempDir())

	builtins := BuiltinDefinitions()
	for name := range builtins {
		def, ok := reg.Get(name)
		if !ok {
			t.Errorf("builtin %q not found in registry", name)
			continue
		}
		if def.Name != name {
			t.Errorf("expected Name=%q, got %q", name, def.Name)
		}
	}
}

func TestRegistry_CustomOverride(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	agentsDir := filepath.Join(projectDir, ".pi-go", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write a custom definition that overrides the builtin "explore" agent.
	customDef := `---
name: explore
description: Custom explorer with extra tools
model: custom-model
tools: read, grep, find, ls, write
max-turns: 20
---

You are a custom exploration agent.
`
	if err := os.WriteFile(filepath.Join(agentsDir, "explore.md"), []byte(customDef), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	reg := NewRegistry(projectDir, t.TempDir())

	def, ok := reg.Get("explore")
	if !ok {
		t.Fatal("explore not found after custom override")
	}
	if def.Description != "Custom explorer with extra tools" {
		t.Errorf("expected custom description, got %q", def.Description)
	}
	if def.Model != "custom-model" {
		t.Errorf("expected model=custom-model, got %q", def.Model)
	}
	if def.MaxTurns != 20 {
		t.Errorf("expected MaxTurns=20, got %d", def.MaxTurns)
	}
}

func TestRegistry_ListAll(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(t.TempDir(), t.TempDir())

	defs := reg.List()
	builtins := BuiltinDefinitions()

	if len(defs) != len(builtins) {
		t.Fatalf("expected %d definitions, got %d", len(builtins), len(defs))
	}

	// Verify all builtins are present.
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	sort.Strings(names)

	expected := make([]string, 0, len(builtins))
	for name := range builtins {
		expected = append(expected, name)
	}
	sort.Strings(expected)

	for i := range expected {
		if names[i] != expected[i] {
			t.Errorf("index %d: expected %q, got %q", i, expected[i], names[i])
		}
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(t.TempDir(), t.TempDir())

	newDef := Definition{
		Name:        "custom_agent",
		Description: "A dynamically registered agent",
		Model:       "fast",
		MaxTurns:    5,
	}
	reg.Register(newDef)

	def, ok := reg.Get("custom_agent")
	if !ok {
		t.Fatal("custom_agent not found after Register")
	}
	if def.Description != "A dynamically registered agent" {
		t.Errorf("unexpected description: %q", def.Description)
	}
}

func TestRegistry_RegisterOverridesExisting(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(t.TempDir(), t.TempDir())

	override := Definition{
		Name:        "explore",
		Description: "Runtime override",
		Model:       "turbo",
		MaxTurns:    99,
	}
	reg.Register(override)

	def, ok := reg.Get("explore")
	if !ok {
		t.Fatal("explore not found after Register override")
	}
	if def.Description != "Runtime override" {
		t.Errorf("expected overridden description, got %q", def.Description)
	}
	if def.MaxTurns != 99 {
		t.Errorf("expected MaxTurns=99, got %d", def.MaxTurns)
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(t.TempDir(), t.TempDir())

	_, ok := reg.Get("nonexistent_agent")
	if ok {
		t.Error("expected ok=false for missing agent")
	}
}
