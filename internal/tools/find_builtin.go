// ABOUTME: Built-in find fallback using filepath.WalkDir and filepath.Match
// ABOUTME: Used when ripgrep is not available on the system

package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findBuiltin searches for files matching a glob pattern using the standard library.
func findBuiltin(pattern, path string) (string, error) {
	var b strings.Builder

	err := filepath.WalkDir(path, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		if matchGlob(filepath.Base(fpath), pattern) {
			fmt.Fprintln(&b, fpath)
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walking %s: %w", path, err)
	}

	if b.Len() == 0 {
		return "no files found", nil
	}
	return b.String(), nil
}
