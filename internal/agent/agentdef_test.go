// ABOUTME: Tests for agent definition parsing and model resolution
// ABOUTME: Covers allowed-tools parsing, model shorthand resolution, round-trip

package agent

import (
	"testing"
)

func TestParseDefinition_AllowedTools(t *testing.T) {
	t.Parallel()

	content := `---
name: reviewer
description: Code review agent
model: default
tools: read, grep
allowed-tools: read, write, edit
max-turns: 10
---

You review code carefully.
`
	def := parseAgentFile(content, "reviewer.md")

	if def.Name != "reviewer" {
		t.Errorf("Name = %q; want %q", def.Name, "reviewer")
	}

	want := []string{"read", "write", "edit"}
	if len(def.AllowedTools) != len(want) {
		t.Fatalf("AllowedTools length = %d; want %d", len(def.AllowedTools), len(want))
	}
	for i, v := range want {
		if def.AllowedTools[i] != v {
			t.Errorf("AllowedTools[%d] = %q; want %q", i, def.AllowedTools[i], v)
		}
	}
}

func TestResolveAgentModel_Fast(t *testing.T) {
	t.Parallel()

	got := ResolveAgentModel("fast")
	want := "claude-haiku-4-5-20251001"
	if got != want {
		t.Errorf("ResolveAgentModel(%q) = %q; want %q", "fast", got, want)
	}
}

func TestResolveAgentModel_Default(t *testing.T) {
	t.Parallel()

	got := ResolveAgentModel("default")
	want := "claude-sonnet-4-6"
	if got != want {
		t.Errorf("ResolveAgentModel(%q) = %q; want %q", "default", got, want)
	}
}

func TestResolveAgentModel_Empty(t *testing.T) {
	t.Parallel()

	got := ResolveAgentModel("")
	want := "claude-sonnet-4-6"
	if got != want {
		t.Errorf("ResolveAgentModel(%q) = %q; want %q", "", got, want)
	}
}

func TestResolveAgentModel_Powerful(t *testing.T) {
	t.Parallel()

	got := ResolveAgentModel("powerful")
	want := "claude-opus-4-6"
	if got != want {
		t.Errorf("ResolveAgentModel(%q) = %q; want %q", "powerful", got, want)
	}
}

func TestResolveAgentModel_Custom(t *testing.T) {
	t.Parallel()

	got := ResolveAgentModel("my-model-v2")
	want := "my-model-v2"
	if got != want {
		t.Errorf("ResolveAgentModel(%q) = %q; want %q", "my-model-v2", got, want)
	}
}

func TestParseDefinition_RoundTrip(t *testing.T) {
	t.Parallel()

	content := `---
name: deployer
description: Deployment specialist
model: powerful
tools: bash, read, ls
disallowed-tools: write, edit
allowed-tools: read, bash
max-turns: 8
---

You deploy applications safely.
`
	def := parseAgentFile(content, "deployer.md")

	if def.Name != "deployer" {
		t.Errorf("Name = %q; want %q", def.Name, "deployer")
	}
	if def.Description != "Deployment specialist" {
		t.Errorf("Description = %q; want %q", def.Description, "Deployment specialist")
	}
	if def.Model != "powerful" {
		t.Errorf("Model = %q; want %q", def.Model, "powerful")
	}
	if def.MaxTurns != 8 {
		t.Errorf("MaxTurns = %d; want %d", def.MaxTurns, 8)
	}

	wantTools := []string{"bash", "read", "ls"}
	if len(def.Tools) != len(wantTools) {
		t.Fatalf("Tools length = %d; want %d", len(def.Tools), len(wantTools))
	}
	for i, v := range wantTools {
		if def.Tools[i] != v {
			t.Errorf("Tools[%d] = %q; want %q", i, def.Tools[i], v)
		}
	}

	wantDisallowed := []string{"write", "edit"}
	if len(def.DisallowedTools) != len(wantDisallowed) {
		t.Fatalf("DisallowedTools length = %d; want %d", len(def.DisallowedTools), len(wantDisallowed))
	}
	for i, v := range wantDisallowed {
		if def.DisallowedTools[i] != v {
			t.Errorf("DisallowedTools[%d] = %q; want %q", i, def.DisallowedTools[i], v)
		}
	}

	wantAllowed := []string{"read", "bash"}
	if len(def.AllowedTools) != len(wantAllowed) {
		t.Fatalf("AllowedTools length = %d; want %d", len(def.AllowedTools), len(wantAllowed))
	}
	for i, v := range wantAllowed {
		if def.AllowedTools[i] != v {
			t.Errorf("AllowedTools[%d] = %q; want %q", i, def.AllowedTools[i], v)
		}
	}

	if def.SystemPrompt != "You deploy applications safely." {
		t.Errorf("SystemPrompt = %q; want %q", def.SystemPrompt, "You deploy applications safely.")
	}
}
