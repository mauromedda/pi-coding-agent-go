// ABOUTME: Tests for version-aware prompt loader with disk/embed fallback
// ABOUTME: Validates composition, overrides, active version, and fragment loading

package prompts

import (
	"os"
	"path/filepath"
	"slices"
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

	// Embedded active.yaml has version: "v1.1.0".
	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	v, err := l.ActiveVersion()
	if err != nil {
		t.Fatalf("ActiveVersion() error = %v", err)
	}
	if v != "v1.1.0" {
		t.Errorf("ActiveVersion() = %q; want %q", v, "v1.1.0")
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

	if !slices.Contains(versions, "v1.0.0") {
		t.Errorf("AvailableVersions() = %v; want to contain v1.0.0", versions)
	}
	if !slices.Contains(versions, "v1.1.0") {
		t.Errorf("AvailableVersions() = %v; want to contain v1.1.0", versions)
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

func TestLoader_Compose_V110_Embedded(t *testing.T) {
	t.Parallel()

	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	vars := map[string]string{
		"MODE": "execute",
		"DATE": "2026-02-23",
		"CWD":  "/workspace",
	}

	got, err := l.Compose("v1.1.0", vars)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	// Verify key phrases from each fragment are present.
	checks := []struct {
		phrase string
		desc   string
	}{
		{"Core Principles", "system.md header"},
		{"pi-go", "system.md identity"},
		{"2026-02-23", "DATE variable substitution"},
		{"Tool Usage Protocol", "core/tool-usage.md"},
		{"Read-Before-Edit", "core/file-operations.md"},
		{"Code Quality", "core/code-quality.md"},
		{"Error Recovery", "core/error-recovery.md"},
		{"Safety", "core/safety.md"},
		{"Git Protocol", "core/git-protocol.md"},
		{"Communication", "core/communication.md"},
		{"EXECUTE Mode", "modes/execute.md"},
	}
	for _, c := range checks {
		if !strings.Contains(got, c.phrase) {
			t.Errorf("missing %s phrase %q in composed output", c.desc, c.phrase)
		}
	}
}

func TestLoader_Compose_V110_AllModes(t *testing.T) {
	t.Parallel()

	modes := []struct {
		name   string
		marker string
	}{
		{"execute", "EXECUTE Mode"},
		{"plan", "PLAN Mode"},
		{"debug", "DEBUG Mode"},
		{"explore", "EXPLORE Mode"},
		{"refactor", "REFACTOR Mode"},
	}

	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()

			vars := map[string]string{
				"MODE": m.name,
				"DATE": "2026-02-23",
				"CWD":  "/workspace",
			}

			got, err := l.Compose("v1.1.0", vars)
			if err != nil {
				t.Fatalf("Compose(mode=%s) error = %v", m.name, err)
			}

			if !strings.Contains(got, m.marker) {
				t.Errorf("Compose(mode=%s) missing marker %q", m.name, m.marker)
			}

			// All modes should include the core fragments.
			if !strings.Contains(got, "Core Principles") {
				t.Errorf("Compose(mode=%s) missing Core Principles from system.md", m.name)
			}
		})
	}
}

func TestLoader_Compose_UsesCache(t *testing.T) {
	t.Parallel()

	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")
	l.Cache = NewCache()

	vars := map[string]string{
		"MODE": "execute",
		"DATE": "2026-02-23",
		"CWD":  "/workspace",
	}

	// First call populates the cache.
	got1, err := l.Compose("v1.0.0", vars)
	if err != nil {
		t.Fatalf("first Compose() error = %v", err)
	}
	if l.Cache.Size() != 1 {
		t.Errorf("cache size = %d after first compose; want 1", l.Cache.Size())
	}

	// Second call with same args should hit cache and return identical result.
	got2, err := l.Compose("v1.0.0", vars)
	if err != nil {
		t.Fatalf("second Compose() error = %v", err)
	}
	if got1 != got2 {
		t.Error("second Compose() returned different result; expected cache hit")
	}

	// Different vars should miss cache.
	vars2 := map[string]string{
		"MODE": "plan",
		"DATE": "2026-02-24",
		"CWD":  "/other",
	}
	got3, err := l.Compose("v1.0.0", vars2)
	if err != nil {
		t.Fatalf("third Compose() error = %v", err)
	}
	if l.Cache.Size() != 2 {
		t.Errorf("cache size = %d after third compose; want 2", l.Cache.Size())
	}
	if got3 == got1 {
		t.Error("different vars should produce different composed output")
	}
}

func TestLoader_Compose_V110_CoreFragments(t *testing.T) {
	t.Parallel()

	fragments := []struct {
		path   string
		marker string
	}{
		{"core/tool-usage.md", "Tool Usage Protocol"},
		{"core/file-operations.md", "Read-Before-Edit"},
		{"core/code-quality.md", "Code Quality"},
		{"core/error-recovery.md", "Error Recovery"},
		{"core/safety.md", "Safety"},
		{"core/git-protocol.md", "Git Protocol"},
		{"core/communication.md", "Communication"},
	}

	l := NewLoader("/nonexistent/prompts", "/nonexistent/overrides")

	for _, f := range fragments {
		t.Run(f.path, func(t *testing.T) {
			t.Parallel()

			data, err := l.LoadFragment("v1.1.0", f.path)
			if err != nil {
				t.Fatalf("LoadFragment(%s) error = %v", f.path, err)
			}

			if !strings.Contains(string(data), f.marker) {
				t.Errorf("LoadFragment(%s) missing marker %q", f.path, f.marker)
			}
		})
	}
}
