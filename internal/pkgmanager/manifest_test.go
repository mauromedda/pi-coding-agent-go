// ABOUTME: Tests for manifest CRUD and file persistence
// ABOUTME: Validates add/remove/find and atomic save/load round-trip

package pkgmanager

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManifest_AddAndFind(t *testing.T) {
	t.Parallel()
	m := &Manifest{}

	info := Info{Name: "lodash", Source: SourceNPM, Version: "4.17.21"}
	m.Add(info)

	found := m.Find("lodash", false)
	if found == nil {
		t.Fatal("expected to find lodash")
	}
	if found.Version != "4.17.21" {
		t.Errorf("Version = %q; want %q", found.Version, "4.17.21")
	}
}

func TestManifest_AddUpdatesExisting(t *testing.T) {
	t.Parallel()
	m := &Manifest{}

	m.Add(Info{Name: "pkg", Version: "1.0"})
	m.Add(Info{Name: "pkg", Version: "2.0"})

	if len(m.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(m.Packages))
	}
	if m.Packages[0].Version != "2.0" {
		t.Errorf("Version = %q; want %q", m.Packages[0].Version, "2.0")
	}
}

func TestManifest_Remove(t *testing.T) {
	t.Parallel()
	m := &Manifest{}
	m.Add(Info{Name: "a"})
	m.Add(Info{Name: "b"})
	m.Add(Info{Name: "c"})

	removed := m.Remove("b", false)
	if !removed {
		t.Error("expected Remove to return true")
	}
	if len(m.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(m.Packages))
	}
	if m.Find("b", false) != nil {
		t.Error("expected b to be removed")
	}
}

func TestManifest_RemoveNotFound(t *testing.T) {
	t.Parallel()
	m := &Manifest{}
	m.Add(Info{Name: "a"})

	if m.Remove("z", false) {
		t.Error("expected Remove to return false for non-existent package")
	}
}

func TestManifest_FindNotFound(t *testing.T) {
	t.Parallel()
	m := &Manifest{}
	if m.Find("nope", false) != nil {
		t.Error("expected nil for non-existent package")
	}
}

func TestManifest_LocalVsGlobal(t *testing.T) {
	t.Parallel()
	m := &Manifest{}
	m.Add(Info{Name: "pkg", Version: "global", Local: false})
	m.Add(Info{Name: "pkg", Version: "local", Local: true})

	if len(m.Packages) != 2 {
		t.Fatalf("expected 2 packages (global + local), got %d", len(m.Packages))
	}

	global := m.Find("pkg", false)
	local := m.Find("pkg", true)

	if global == nil || global.Version != "global" {
		t.Error("expected global version")
	}
	if local == nil || local.Version != "local" {
		t.Error("expected local version")
	}
}

func TestLoadSaveManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := &Manifest{}
	original.Add(Info{
		Name:        "test-pkg",
		Source:      SourceGit,
		Path:        "https://github.com/user/repo",
		Version:     "v1.0",
		InstalledAt: time.Now().Truncate(time.Second),
	})

	if err := SaveManifest(dir, original); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if len(loaded.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(loaded.Packages))
	}
	if loaded.Packages[0].Name != "test-pkg" {
		t.Errorf("Name = %q; want %q", loaded.Packages[0].Name, "test-pkg")
	}
	if loaded.Packages[0].Version != "v1.0" {
		t.Errorf("Version = %q; want %q", loaded.Packages[0].Version, "v1.0")
	}
}

func TestLoadManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Packages) != 0 {
		t.Errorf("expected empty manifest, got %d packages", len(m.Packages))
	}
}

func TestLoadManifest_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, manifestFileName), []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(dir)
	if err == nil {
		t.Error("expected error for corrupt manifest")
	}
}

func TestSaveManifest_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")

	m := &Manifest{}
	m.Add(Info{Name: "pkg"})

	if err := SaveManifest(dir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, manifestFileName)); err != nil {
		t.Fatalf("manifest file not created: %v", err)
	}
}
