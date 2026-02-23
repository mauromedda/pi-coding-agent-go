// ABOUTME: Tests for profile loading, validation, and default trait sets
// ABOUTME: Covers YAML parsing, directory scanning, and field validation

package personality

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultTraitSet(t *testing.T) {
	t.Parallel()
	ts := DefaultTraitSet()
	if ts.Verbosity != "balanced" {
		t.Errorf("Verbosity = %q; want %q", ts.Verbosity, "balanced")
	}
	if ts.RiskTolerance != "balanced" {
		t.Errorf("RiskTolerance = %q; want %q", ts.RiskTolerance, "balanced")
	}
	if ts.Explanation != "standard" {
		t.Errorf("Explanation = %q; want %q", ts.Explanation, "standard")
	}
	if ts.AutoPlan != "suggest" {
		t.Errorf("AutoPlan = %q; want %q", ts.AutoPlan, "suggest")
	}
}

func TestLoadProfile_Valid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `name: test-profile
description: A test profile
traits:
  verbosity: verbose
  risk_tolerance: cautious
  explanation: detailed
  auto_plan: auto
checks:
  security: strict
  quality: standard
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if p.Name != "test-profile" {
		t.Errorf("Name = %q; want %q", p.Name, "test-profile")
	}
	if p.Traits.Verbosity != "verbose" {
		t.Errorf("Traits.Verbosity = %q; want %q", p.Traits.Verbosity, "verbose")
	}
	if p.Checks["security"] != "strict" {
		t.Errorf("Checks[security] = %q; want %q", p.Checks["security"], "strict")
	}
}

func TestLoadProfile_InvalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProfile(path)
	if err == nil {
		t.Error("LoadProfile() expected error for invalid YAML; got nil")
	}
}

func TestLoadProfile_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := LoadProfile("/nonexistent/path/profile.yaml")
	if err == nil {
		t.Error("LoadProfile() expected error for missing file; got nil")
	}
}

func TestLoadProfiles_MultipleFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	profiles := map[string]string{
		"alpha.yaml": `name: alpha
description: Alpha profile
traits:
  verbosity: terse
  risk_tolerance: aggressive
  explanation: minimal
  auto_plan: never
checks:
  security: minimal
`,
		"beta.yaml": `name: beta
description: Beta profile
traits:
  verbosity: verbose
  risk_tolerance: cautious
  explanation: detailed
  auto_plan: auto
checks:
  quality: strict
`,
	}

	for fname, content := range profiles {
		path := filepath.Join(dir, fname)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Also add a non-yaml file that should be ignored
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadProfiles(dir)
	if err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d; want 2", len(result))
	}
	if result["alpha"] == nil {
		t.Error("result[\"alpha\"] is nil")
	}
	if result["beta"] == nil {
		t.Error("result[\"beta\"] is nil")
	}
}

func TestProfile_Validate_Valid(t *testing.T) {
	t.Parallel()
	p := &Profile{
		Name:        "valid",
		Description: "A valid profile",
		Traits:      DefaultTraitSet(),
		Checks:      map[string]string{"security": "standard"},
	}
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() error = %v; want nil", err)
	}
}

func TestProfile_Validate_InvalidVerbosity(t *testing.T) {
	t.Parallel()
	p := &Profile{
		Name:   "bad",
		Traits: TraitSet{Verbosity: "extreme", RiskTolerance: "balanced", Explanation: "standard", AutoPlan: "suggest"},
		Checks: map[string]string{},
	}
	if err := p.Validate(); err == nil {
		t.Error("Validate() expected error for invalid verbosity; got nil")
	}
}

func TestProfile_Validate_InvalidRiskTolerance(t *testing.T) {
	t.Parallel()
	p := &Profile{
		Name:   "bad",
		Traits: TraitSet{Verbosity: "balanced", RiskTolerance: "yolo", Explanation: "standard", AutoPlan: "suggest"},
		Checks: map[string]string{},
	}
	if err := p.Validate(); err == nil {
		t.Error("Validate() expected error for invalid risk_tolerance; got nil")
	}
}

func TestProfile_Validate_MissingName(t *testing.T) {
	t.Parallel()
	p := &Profile{
		Traits: DefaultTraitSet(),
		Checks: map[string]string{},
	}
	if err := p.Validate(); err == nil {
		t.Error("Validate() expected error for missing name; got nil")
	}
}

func TestProfile_Validate_InvalidCheckLevel(t *testing.T) {
	t.Parallel()
	p := &Profile{
		Name:   "bad",
		Traits: DefaultTraitSet(),
		Checks: map[string]string{"security": "ultra"},
	}
	if err := p.Validate(); err == nil {
		t.Error("Validate() expected error for invalid check level; got nil")
	}
}
