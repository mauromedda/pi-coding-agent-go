// ABOUTME: AutoMemory manages per-project memory entries stored as individual .md files
// ABOUTME: Provides save, load, delete, and list operations with sanitized key names

package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// validKey matches keys containing only alphanumeric, dash, or underscore characters.
var validKey = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// AutoMem manages automatically saved memory entries.
// Entries are stored as individual files in the auto-memory directory.
type AutoMem struct {
	dir string // e.g. ~/.pi-go/memory/auto/
}

// NewAutoMemory creates an AutoMem that persists entries under dir.
// The directory is created if it does not exist.
func NewAutoMemory(dir string) *AutoMem {
	_ = os.MkdirAll(dir, 0o755)
	return &AutoMem{dir: dir}
}

// Save writes content to <dir>/<key>.md.
// The key is validated: only alphanumeric, dash, and underscore are allowed.
func (am *AutoMem) Save(key, content string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	path := filepath.Join(am.dir, key+".md")
	return os.WriteFile(path, []byte(content), 0o644)
}

// Load reads all .md files from the directory and returns them as Entry values
// with Level set to AutoMemory.
func (am *AutoMem) Load() ([]Entry, error) {
	dirEntries, err := os.ReadDir(am.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading auto-memory dir: %w", err)
	}

	var entries []Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		path := filepath.Join(am.dir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
			Source:  path,
			Content: string(data),
			Level:   AutoMemory,
		})
	}
	return entries, nil
}

// Delete removes the file for the given key.
func (am *AutoMem) Delete(key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	path := filepath.Join(am.dir, key+".md")
	return os.Remove(path)
}

// List returns a sorted slice of stored keys (filenames without the .md suffix).
func (am *AutoMem) List() ([]string, error) {
	dirEntries, err := os.ReadDir(am.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing auto-memory dir: %w", err)
	}

	var keys []string
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		key := strings.TrimSuffix(de.Name(), ".md")
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

// validateKey rejects keys that could escape the directory or contain unsafe characters.
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("auto-memory key must not be empty")
	}
	if key == "." || key == ".." {
		return fmt.Errorf("auto-memory key %q is not allowed", key)
	}
	if !validKey.MatchString(key) {
		return fmt.Errorf("auto-memory key %q contains invalid characters; only alphanumeric, dash, and underscore are allowed", key)
	}
	return nil
}
