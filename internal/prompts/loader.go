// ABOUTME: Version-aware prompt loader with disk-first, embed fallback strategy
// ABOUTME: Loads fragments from prompts/ dir on disk or embedded templates/

package prompts

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader loads and composes prompt fragments from versioned directories.
type Loader struct {
	diskDir   string // path to disk prompts/ dir (may not exist)
	embedded  fs.FS  // embedded templates
	overrides string // path to overrides/ dir on disk
	Cache     *Cache // optional cache for composed prompts
}

// activeConfig holds the active version pointer.
type activeConfig struct {
	Version string `yaml:"version"`
}

// NewLoader creates a loader that checks disk first, then falls back to embedded.
// diskDir: path to prompts/ directory (e.g., "./prompts")
// overridesDir: path to overrides directory (e.g., "./prompts/overrides")
func NewLoader(diskDir, overridesDir string) *Loader {
	sub, err := fs.Sub(embeddedFS, "templates")
	if err != nil {
		// Should never happen with valid embed; embedded FS always has "templates".
		panic(fmt.Sprintf("embedded templates sub-fs: %v", err))
	}
	return &Loader{
		diskDir:   diskDir,
		embedded:  sub,
		overrides: overridesDir,
	}
}

// ActiveVersion reads the active version from active.yaml.
// Checks disk first, then embedded.
func (l *Loader) ActiveVersion() (string, error) {
	data, err := l.readFile("active.yaml")
	if err != nil {
		return "", fmt.Errorf("read active.yaml: %w", err)
	}

	var cfg activeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse active.yaml: %w", err)
	}
	return cfg.Version, nil
}

// LoadFragment loads a single fragment file for a given version.
// Precedence: overrides dir -> disk version dir -> embedded version dir.
func (l *Loader) LoadFragment(version, path string) ([]byte, error) {
	// 1. Overrides dir (version-agnostic).
	if l.overrides != "" {
		overridePath := filepath.Join(l.overrides, path)
		if data, err := os.ReadFile(overridePath); err == nil {
			return data, nil
		}
	}

	// 2. Disk version dir.
	if l.diskDir != "" {
		diskPath := filepath.Join(l.diskDir, version, path)
		if data, err := os.ReadFile(diskPath); err == nil {
			return data, nil
		}
	}

	// 3. Embedded version dir.
	embeddedPath := version + "/" + path
	data, err := fs.ReadFile(l.embedded, embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("load fragment %s/%s: not found in overrides, disk, or embedded: %w", version, path, err)
	}
	return data, nil
}

// Compose assembles the full system prompt from a manifest and variables.
// It loads the manifest, resolves composition_order paths (substituting variables),
// loads each fragment, renders template variables, and concatenates.
// If a Cache is set, results are cached by version+vars to skip repeated file I/O.
func (l *Loader) Compose(version string, vars map[string]string) (string, error) {
	// Check cache first.
	if l.Cache != nil {
		if cached, ok := l.Cache.Get(version, vars); ok {
			return cached, nil
		}
	}

	// Load the manifest.
	manifestData, err := l.LoadFragment(version, "manifest.yaml")
	if err != nil {
		return "", fmt.Errorf("load manifest: %w", err)
	}

	manifest, err := LoadManifest(manifestData)
	if err != nil {
		return "", err
	}

	// Merge manifest default variables with provided vars (provided wins).
	merged := make(map[string]string, len(manifest.Variables)+len(vars))
	maps.Copy(merged, manifest.Variables)
	maps.Copy(merged, vars)

	// Compose fragments in order.
	var parts []string
	for _, pathTemplate := range manifest.CompositionOrder {
		// Resolve path-level variables like {{MODE}}.
		resolvedPath := pathTemplate
		for k, v := range merged {
			resolvedPath = strings.ReplaceAll(resolvedPath, "{{"+k+"}}", v)
		}

		fragment, err := l.LoadFragment(version, resolvedPath)
		if err != nil {
			return "", fmt.Errorf("compose fragment %q: %w", resolvedPath, err)
		}

		rendered, err := RenderVariables(string(fragment), merged)
		if err != nil {
			return "", fmt.Errorf("render fragment %q: %w", resolvedPath, err)
		}

		parts = append(parts, rendered)
	}

	result := strings.Join(parts, "\n")

	// Store in cache for next call.
	if l.Cache != nil {
		l.Cache.Set(version, vars, result)
	}

	return result, nil
}

// AvailableVersions returns all version directories found in embedded and disk dirs.
func (l *Loader) AvailableVersions() ([]string, error) {
	seen := make(map[string]bool)

	// Embedded versions.
	entries, err := fs.ReadDir(l.embedded, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded versions: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "v") {
			seen[e.Name()] = true
		}
	}

	// Disk versions.
	if l.diskDir != "" {
		diskEntries, err := os.ReadDir(l.diskDir)
		if err == nil {
			for _, e := range diskEntries {
				if e.IsDir() && strings.HasPrefix(e.Name(), "v") {
					seen[e.Name()] = true
				}
			}
		}
	}

	versions := make([]string, 0, len(seen))
	for v := range seen {
		versions = append(versions, v)
	}
	return versions, nil
}

// readFile reads a file from disk first, then embedded.
func (l *Loader) readFile(path string) ([]byte, error) {
	if l.diskDir != "" {
		diskPath := filepath.Join(l.diskDir, path)
		if data, err := os.ReadFile(diskPath); err == nil {
			return data, nil
		}
	}

	data, err := fs.ReadFile(l.embedded, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: not found on disk or embedded: %w", path, err)
	}
	return data, nil
}
