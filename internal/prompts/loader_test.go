// ABOUTME: Tests for version-aware prompt loader with disk/embed fallback
// ABOUTME: Validates composition, overrides, active version, and fragment loading

package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoader_Compose_Embedded(t *testing.T) {
	t.Parallel()

	// No disk dir: uses embedded templates only.
	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	vars := map[string]string{
		"MODE": "execute",
		"DATE": "2026-02-23",
		"CWD":  "/workspace",
	}

	got, err := l.Compose("v1.0.0", vars)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if !strings.Contains(got, "pi-go") {
		t.Errorf("expected system prompt header; got %q", got)
	}
	if !strings.Contains(got, "2026-02-23") {
		t.Errorf("expected DATE variable substitution; got %q", got)
	}
	if !strings.Contains(got, "EXECUTE mode") {
		t.Errorf("expected execute mode content; got %q", got)
	}
}

func TestLoader_Compose_DiskOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	versionDir := filepath.Join(dir, "v1.0.0", "modes")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a custom system.md to disk.
	if err := os.WriteFile(
		filepath.Join(dir, "v1.0.0", "system.md"),
		[]byte("Custom system: {{.DATE}}"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Copy manifest so loader can read it.
	manifest := `version: "v1.0.0"
description: "disk override"
compatible_models: ["claude-*"]
composition_order:
  - "system.md"
  - "modes/{{MODE}}.md"
variables:
  MODE: "execute"
`
	if err := os.WriteFile(
		filepath.Join(dir, "v1.0.0", "manifest.yaml"),
		[]byte(manifest),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir, "/nonexistent/overrides")

	vars := map[string]string{
		"MODE": "execute",
		"DATE": "2026-01-01",
	}

	got, err := l.Compose("v1.0.0", vars)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if !strings.Contains(got, "Custom system: 2026-01-01") {
		t.Errorf("expected disk override content; got %q", got)
	}
}

func TestLoader_Compose_OverridesDir(t *testing.T) {
	t.Parallel()

	overridesDir := t.TempDir()
	modesDir := filepath.Join(overridesDir, "modes")
	if err := os.MkdirAll(modesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Override just the execute mode fragment.
	if err := os.WriteFile(
		filepath.Join(modesDir, "execute.md"),
		[]byte("OVERRIDDEN EXECUTE MODE"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	l := NewLoader("/nonexistent/prompts", overridesDir)

	vars := map[string]string{
		"MODE": "execute",
		"DATE": "2026-02-23",
		"CWD":  "/workspace",
	}

	got, err := l.Compose("v1.0.0", vars)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if !strings.Contains(got, "OVERRIDDEN EXECUTE MODE") {
		t.Errorf("expected overrides dir content; got %q", got)
	}
}

func TestLoader_ActiveVersion(t *testing.T) {
	t.Parallel()

	// Embedded active.yaml has version: "v1.0.0".
	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	v, err := l.ActiveVersion()
	if err != nil {
		t.Fatalf("ActiveVersion() error = %v", err)
	}
	if v != "v1.0.0" {
		t.Errorf("ActiveVersion() = %q; want %q", v, "v1.0.0")
	}
}

func TestLoader_AvailableVersions(t *testing.T) {
	t.Parallel()

	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	versions, err := l.AvailableVersions()
	if err != nil {
		t.Fatalf("AvailableVersions() error = %v", err)
	}
	if len(versions) == 0 {
		t.Fatal("AvailableVersions() returned empty; want at least v1.0.0")
	}

	found := false
	for _, v := range versions {
		if v == "v1.0.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("AvailableVersions() = %v; want to contain v1.0.0", versions)
	}
}

func TestLoader_LoadFragment_FallbackChain(t *testing.T) {
	t.Parallel()

	// Test 1: embedded fallback (no disk, no overrides).
	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	data, err := l.LoadFragment("v1.0.0", "system.md")
	if err != nil {
		t.Fatalf("LoadFragment(embedded) error = %v", err)
	}
	if !strings.Contains(string(data), "pi-go") {
		t.Errorf("embedded fragment missing expected content; got %q", string(data))
	}

	// Test 2: disk overrides embedded.
	diskDir := t.TempDir()
	vDir := filepath.Join(diskDir, "v1.0.0")
	if err := os.MkdirAll(vDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vDir, "system.md"), []byte("DISK VERSION"), 0o644); err != nil {
		t.Fatal(err)
	}

	l2 := NewLoader(diskDir, "/nonexistent/overrides")
	data2, err := l2.LoadFragment("v1.0.0", "system.md")
	if err != nil {
		t.Fatalf("LoadFragment(disk) error = %v", err)
	}
	if string(data2) != "DISK VERSION" {
		t.Errorf("disk fragment = %q; want %q", string(data2), "DISK VERSION")
	}

	// Test 3: overrides dir takes precedence over disk and embedded.
	overDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(overDir, "system.md"), []byte("OVERRIDE VERSION"), 0o644); err != nil {
		t.Fatal(err)
	}

	l3 := NewLoader(diskDir, overDir)
	data3, err := l3.LoadFragment("v1.0.0", "system.md")
	if err != nil {
		t.Fatalf("LoadFragment(override) error = %v", err)
	}
	if string(data3) != "OVERRIDE VERSION" {
		t.Errorf("override fragment = %q; want %q", string(data3), "OVERRIDE VERSION")
	}
}
