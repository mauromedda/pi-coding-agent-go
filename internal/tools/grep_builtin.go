// ABOUTME: Built-in grep fallback using stdlib regexp and filepath.WalkDir
// ABOUTME: Used when ripgrep is not available on the system

package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxMatches = 10000

// errMatchLimitReached is a sentinel error used to stop walking early.
var errMatchLimitReached = fmt.Errorf("match limit reached (%d)", maxMatches)

// grepBuiltin searches for pattern matches across files using the standard library.
// Walks the directory tree at path, optionally filtering by an include glob.
// Stops after maxMatches results to prevent unbounded output.
func grepBuiltin(pattern, path, include string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("compiling pattern %q: %w", pattern, err)
	}

	var b strings.Builder
	matchCount := 0
	walkErr := filepath.WalkDir(path, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		if include != "" && !matchGlob(filepath.Base(fpath), include) {
			return nil
		}
		return grepFile(re, fpath, &b, &matchCount)
	})

	if walkErr != nil && walkErr != errMatchLimitReached {
		return "", fmt.Errorf("walking %s: %w", path, walkErr)
	}

	if b.Len() == 0 {
		return "no matches found", nil
	}
	if matchCount >= maxMatches {
		b.WriteString(fmt.Sprintf("\n... [truncated: %d matches shown, limit reached]\n", maxMatches))
	}
	return b.String(), nil
}

// grepFile scans a single file for regex matches, appending results to b.
// Returns errMatchLimitReached if matchCount exceeds maxMatches.
func grepFile(re *regexp.Regexp, path string, b *strings.Builder, matchCount *int) error {
	f, err := os.Open(path)
	if err != nil {
		return nil // skip unreadable files
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			fmt.Fprintf(b, "%s:%d:%s\n", path, lineNum, line)
			*matchCount++
			if *matchCount >= maxMatches {
				return errMatchLimitReached
			}
		}
	}
	return nil
}

// matchGlob checks if name matches the given glob pattern.
func matchGlob(name, pattern string) bool {
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}
