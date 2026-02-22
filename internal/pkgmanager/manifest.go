// ABOUTME: Manifest tracks installed packages in a JSON file
// ABOUTME: Provides CRUD operations for package entries with atomic file writes

package pkgmanager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const manifestFileName = "manifest.json"

// LoadManifest reads a manifest from the given directory.
// Returns an empty manifest if the file does not exist.
func LoadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, manifestFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Manifest{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}

// SaveManifest writes a manifest to the given directory atomically.
func SaveManifest(dir string, m *Manifest) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating manifest directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	path := filepath.Join(dir, manifestFileName)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writing temp manifest: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp manifest: %w", err)
	}

	return nil
}

// Add adds or updates a package entry in the manifest.
func (m *Manifest) Add(info Info) {
	for i, p := range m.Packages {
		if p.Name == info.Name && p.Local == info.Local {
			m.Packages[i] = info
			return
		}
	}
	m.Packages = append(m.Packages, info)
}

// Remove removes a package by name and scope. Returns true if found.
func (m *Manifest) Remove(name string, local bool) bool {
	for i, p := range m.Packages {
		if p.Name == name && p.Local == local {
			m.Packages = append(m.Packages[:i], m.Packages[i+1:]...)
			return true
		}
	}
	return false
}

// Find returns a package by name and scope, or nil if not found.
func (m *Manifest) Find(name string, local bool) *Info {
	for i, p := range m.Packages {
		if p.Name == name && p.Local == local {
			return &m.Packages[i]
		}
	}
	return nil
}
