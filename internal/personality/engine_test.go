// ABOUTME: Tests for the personality engine orchestrating profiles and checks
// ABOUTME: Covers default profiles, profile switching, and prompt composition

package personality

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/personality/checks"
)

func TestNewEngine_DefaultProfiles(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	names := e.ProfileNames()
	if len(names) != 5 {
		t.Errorf("len(ProfileNames()) = %d; want 5", len(names))
	}

	expected := map[string]bool{
		"base":             true,
		"security-focused": true,
		"speed-focused":    true,
		"mentor":           true,
		"architect":        true,
	}
	for _, n := range names {
		if !expected[n] {
			t.Errorf("unexpected profile name: %q", n)
		}
	}
}

func TestEngine_SetProfile_Valid(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	if err := e.SetProfile("security-focused"); err != nil {
		t.Errorf("SetProfile(\"security-focused\") error = %v", err)
	}
	if got := e.ActiveProfile().Name; got != "security-focused" {
		t.Errorf("ActiveProfile().Name = %q; want %q", got, "security-focused")
	}
}

func TestEngine_SetProfile_Invalid(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	if err := e.SetProfile("nonexistent"); err == nil {
		t.Error("SetProfile(\"nonexistent\") expected error; got nil")
	}
}

func TestEngine_ActiveProfile_Default(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	p := e.ActiveProfile()
	if p == nil {
		t.Fatal("ActiveProfile() = nil; want non-nil")
	}
	if p.Name != "base" {
		t.Errorf("ActiveProfile().Name = %q; want %q", p.Name, "base")
	}
}

func TestEngine_ComposePrompt_Base(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	ctx := checks.CheckContext{
		FilesChanged:   3,
		LinesChanged:   100,
		HasTests:       true,
		HasErrorHandling: true,
	}

	prompt := e.ComposePrompt(ctx)
	if prompt == "" {
		t.Error("ComposePrompt() returned empty string")
	}
	// Base profile should produce some instructions
	if !strings.Contains(prompt, "security") && !strings.Contains(prompt, "quality") {
		t.Errorf("ComposePrompt() = %q; expected to contain check-related content", prompt)
	}
}

func TestEngine_ComposePrompt_SecurityFocused(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	if err := e.SetProfile("security-focused"); err != nil {
		t.Fatal(err)
	}

	ctx := checks.CheckContext{
		FilesChanged:      5,
		IsSecurityRelated: true,
	}

	prompt := e.ComposePrompt(ctx)
	if !strings.Contains(strings.ToLower(prompt), "security") {
		t.Errorf("security-focused prompt should contain security instructions; got %q", prompt)
	}
	// Paranoid security should have threat model references
	if !strings.Contains(strings.ToLower(prompt), "threat") {
		t.Errorf("security-focused prompt should contain threat modeling; got %q", prompt)
	}
}

func TestEngine_ComposeTraitInstructions_AcceptsTraitSetParam(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	// Call composeTraitInstructions directly with a custom TraitSet to verify
	// it uses the parameter, not e.active.Traits.
	custom := TraitSet{
		Verbosity:     "terse",
		RiskTolerance: "aggressive",
		Explanation:   "detailed",
		AutoPlan:      "never",
	}

	result := e.composeTraitInstructions(custom)
	if !strings.Contains(result, "concise") {
		t.Errorf("expected terse instruction containing 'concise'; got %q", result)
	}
	if !strings.Contains(result, "speed and pragmatism") {
		t.Errorf("expected aggressive risk instruction; got %q", result)
	}
	if !strings.Contains(result, "reasoning behind every") {
		t.Errorf("expected detailed explanation instruction; got %q", result)
	}
	if !strings.Contains(result, "Do not generate plans") {
		t.Errorf("expected never auto_plan instruction; got %q", result)
	}
}

func TestEngine_ProfileNames(t *testing.T) {
	t.Parallel()
	e, err := NewEngine("")
	if err != nil {
		t.Fatal(err)
	}

	names := e.ProfileNames()
	if len(names) == 0 {
		t.Error("ProfileNames() returned empty slice")
	}

	// Verify sorted or at least deterministic
	seen := make(map[string]bool)
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate profile name: %q", n)
		}
		seen[n] = true
	}
}
