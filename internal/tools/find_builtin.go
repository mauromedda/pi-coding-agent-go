// ABOUTME: Built-in find fallback using filepath.WalkDir with ** glob support
// ABOUTME: Sorts results by modification time (newest first), supports head_limit

package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileEntry holds a file path and its modification time for sorting.
type fileEntry struct {
	Path    string
	ModTime int64
}

// findBuiltin searches for files matching a glob pattern using the standard library.
// Results are sorted by modification time (newest first).
func findBuiltin(pattern, path string, headLimit int) (string, error) {
	hasDoubleStar := strings.Contains(pattern, "**")
	var entries []fileEntry

	err := filepath.WalkDir(path, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Use relative path for pattern matching so "sub/**/*.go" works
		relPath, relErr := filepath.Rel(path, fpath)
		if relErr != nil {
			relPath = fpath
		}

		var matched bool
		if hasDoubleStar {
			matched = matchDoubleStarGlob(relPath, pattern)
		} else {
			matched = matchGlob(filepath.Base(relPath), pattern)
		}

		if matched {
			info, statErr := d.Info()
			if statErr != nil {
				return nil
			}
			entries = append(entries, fileEntry{
				Path:    fpath,
				ModTime: info.ModTime().UnixNano(),
			})
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walking %s: %w", path, err)
	}

	if len(entries) == 0 {
		return "no files found", nil
	}

	// Sort by modification time, newest first
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ModTime > entries[j].ModTime
	})

	if headLimit > 0 && headLimit < len(entries) {
		entries = entries[:headLimit]
	}

	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintln(&b, e.Path)
	}
	return b.String(), nil
}
