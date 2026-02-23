// ABOUTME: Memory hierarchy loading with 5-level resolution and @import expansion
// ABOUTME: Loads CLAUDE.md, rules dirs, and auto-memory

package memory

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"
)

// Level represents the priority of a memory entry.
// Lower values are loaded first (higher priority in prompt ordering).
type Level int

const (
	ProjectRules     Level = iota // .pi-go/rules/*.md
	ClaudeCompat                  // ./CLAUDE.md or ./.claude/CLAUDE.md
	ClaudeRules                   // .claude/rules/*.md
	UserClaudeCompat              // ~/.claude/CLAUDE.md
	AutoMemory                    // ~/.pi-go/projects/<sha256>/memory/
)

const maxImportDepth = 5

// Entry represents a single loaded memory file.
type Entry struct {
	Source  string // File path
	Content string // Resolved content (imports expanded)
	Level   Level
	Paths   []string // Glob patterns for path-specific rules
}

// levelCount is the number of memory hierarchy levels.
const levelCount = 5

// Load reads memory entries from all 5 levels in parallel, returning them sorted by level.
// Each level writes to its own slot in a fixed array; no mutex is needed.
func Load(projectDir, homeDir string) ([]Entry, error) {
	var levels [levelCount][]Entry

	var g errgroup.Group

	// Level 0: Project rules (.pi-go/rules/*.md)
	g.Go(func() error {
		rulesDir := filepath.Join(projectDir, ".pi-go", "rules")
		if entries, err := loadRulesDir(rulesDir, ProjectRules); err == nil {
			levels[0] = entries
		}
		return nil
	})

	// Level 1: Claude compat project (CLAUDE.md or .claude/CLAUDE.md)
	g.Go(func() error {
		if e, ok := loadFirstFile(projectDir, ClaudeCompat,
			filepath.Join(projectDir, "CLAUDE.md"),
			filepath.Join(projectDir, ".claude", "CLAUDE.md"),
		); ok {
			levels[1] = []Entry{e}
		}
		return nil
	})

	// Level 2: Claude rules (.claude/rules/*.md)
	g.Go(func() error {
		claudeRulesDir := filepath.Join(projectDir, ".claude", "rules")
		if entries, err := loadRulesDir(claudeRulesDir, ClaudeRules); err == nil {
			levels[2] = entries
		}
		return nil
	})

	// Level 3: User Claude compat (~/.claude/CLAUDE.md)
	g.Go(func() error {
		if e, ok := loadSingleFile(filepath.Join(homeDir, ".claude", "CLAUDE.md"), UserClaudeCompat); ok {
			levels[3] = []Entry{e}
		}
		return nil
	})

	// Level 4: Auto-memory
	g.Go(func() error {
		autoDir := AutoMemoryDir(projectDir, homeDir)
		if entries, err := loadRulesDir(autoDir, AutoMemory); err == nil {
			levels[4] = entries
		}
		return nil
	})

	_ = g.Wait() // individual goroutines never return errors

	// Flatten in level order (already sorted by array index)
	var entries []Entry
	for _, levelEntries := range levels {
		entries = append(entries, levelEntries...)
	}

	return entries, nil
}

// FormatForPrompt renders entries as a system prompt section.
// If activeFiles is non-nil, path-filtered entries are only included when
// at least one active file matches their glob patterns.
func FormatForPrompt(entries []Entry, activeFiles []string) string {
	if len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	for _, e := range entries {
		// Skip path-filtered entries that don't match active files
		if len(e.Paths) > 0 && len(activeFiles) > 0 {
			if !matchAnyFile(e.Paths, activeFiles) {
				continue
			}
		}
		// If paths are set but no active files provided, skip
		if len(e.Paths) > 0 && len(activeFiles) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("# Memory: %s\n%s\n\n", filepath.Base(e.Source), e.Content))
	}
	return b.String()
}

// AutoMemoryDir returns the auto-memory directory for a project.
// Uses SHA-256 of the project path for uniqueness.
func AutoMemoryDir(projectDir, homeDir string) string {
	h := sha256.Sum256([]byte(projectDir))
	hash := fmt.Sprintf("%x", h)[:16]
	return filepath.Join(homeDir, ".pi-go", "projects", hash, "memory")
}

// expandImports resolves @path references in content.
// Tracks visited files for cycle detection and enforces max depth.
func expandImports(content, baseDir string, visited map[string]bool, depth int) (string, error) {
	if depth > maxImportDepth {
		return "", fmt.Errorf("import depth exceeds maximum (%d)", maxImportDepth)
	}
	if visited == nil {
		visited = make(map[string]bool)
	}

	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "@@") {
			result = append(result, line)
			continue
		}

		importPath := strings.TrimPrefix(trimmed, "@")
		if importPath == "" {
			result = append(result, line)
			continue
		}

		absPath := importPath
		if !filepath.IsAbs(importPath) {
			absPath = filepath.Join(baseDir, importPath)
		}
		absPath, _ = filepath.Abs(absPath)

		if visited[absPath] {
			return "", fmt.Errorf("import cycle detected: %s", absPath)
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			// Missing files: insert comment noting the missing import
			result = append(result, fmt.Sprintf("<!-- import not found: %s -->", importPath))
			continue
		}

		visited[absPath] = true
		expanded, err := expandImports(string(data), filepath.Dir(absPath), visited, depth+1)
		if err != nil {
			return "", err
		}
		result = append(result, expanded)
	}

	return strings.Join(result, "\n"), nil
}

// parseFrontmatter extracts YAML-like frontmatter and returns body + paths.
func parseFrontmatter(content string) (string, []string) {
	if !strings.HasPrefix(content, "---\n") {
		return content, nil
	}

	endIdx := strings.Index(content[4:], "\n---")
	if endIdx < 0 {
		return content, nil
	}

	fm := content[4 : 4+endIdx]
	body := strings.TrimSpace(content[4+endIdx+4:])

	var paths []string
	for line := range strings.SplitSeq(fm, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "paths" {
			paths = parsePaths(value)
		}
	}

	return body, paths
}

// parsePaths handles both ["*.go", "*.mod"] and *.go, *.mod syntax.
func parsePaths(value string) []string {
	// Strip brackets
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")

	parts := strings.Split(value, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "\"'")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// matchPath checks if any pattern matches the given path.
// Tries matching against both the full path and the base name.
func matchPath(patterns []string, path string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// matchAnyFile returns true if any activeFile matches any pattern.
func matchAnyFile(patterns []string, files []string) bool {
	for _, f := range files {
		if matchPath(patterns, f) {
			return true
		}
	}
	return false
}

// loadFirstFile tries paths in order; returns the first that exists.
func loadFirstFile(baseDir string, level Level, paths ...string) (Entry, bool) {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content, err := expandImports(string(data), filepath.Dir(p), nil, 0)
		if err != nil {
			content = string(data) // Fall back to raw content on expand error
		}
		return Entry{
			Source:  p,
			Content: content,
			Level:   level,
		}, true
	}
	return Entry{}, false
}

// loadSingleFile loads a single file as an entry.
func loadSingleFile(path string, level Level) (Entry, bool) {
	return loadFirstFile(filepath.Dir(path), level, path)
}

// loadRulesDir loads all .md files from a rules directory.
func loadRulesDir(dir string, level Level) ([]Entry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		raw := string(data)
		body, paths := parseFrontmatter(raw)

		content, err := expandImports(body, dir, nil, 0)
		if err != nil {
			content = body
		}

		entries = append(entries, Entry{
			Source:  path,
			Content: content,
			Level:   level,
			Paths:   paths,
		})
	}

	return entries, nil
}
