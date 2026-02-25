// ABOUTME: Built-in grep fallback using stdlib regexp and filepath.WalkDir
// ABOUTME: Supports all output modes, context lines, case-insensitive, multiline, glob/type filtering

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

// grepBuiltin searches for pattern matches using the standard library.
func grepBuiltin(opts grepOptions) (string, error) {
	mode := opts.effectiveOutputMode()
	pattern := opts.Pattern

	if opts.Insensitive {
		pattern = "(?i)" + pattern
	}
	if opts.Multiline {
		pattern = "(?s)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("compiling pattern %q: %w", opts.Pattern, err)
	}

	if opts.Multiline {
		return grepMultiline(re, opts, mode)
	}

	var entries []string
	matchCount := 0

	walkErr := filepath.WalkDir(opts.Path, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesGrepFilter(fpath, opts) {
			return nil
		}

		switch mode {
		case "files_with_matches":
			matched, scanErr := fileHasMatch(re, fpath)
			if scanErr != nil {
				return nil
			}
			if matched {
				entries = append(entries, fpath)
				matchCount++
			}
		case "count":
			count, scanErr := countFileMatches(re, fpath)
			if scanErr != nil {
				return nil
			}
			if count > 0 {
				entries = append(entries, fmt.Sprintf("%s:%d", fpath, count))
				matchCount++
			}
		default: // "content"
			fileEntries, scanErr := grepFileContent(re, fpath, opts)
			if scanErr != nil {
				return nil
			}
			entries = append(entries, fileEntries...)
			matchCount += len(fileEntries)
		}

		if matchCount >= maxMatches {
			return errMatchLimitReached
		}
		return nil
	})

	if walkErr != nil && walkErr != errMatchLimitReached {
		return "", fmt.Errorf("walking %s: %w", opts.Path, walkErr)
	}

	if len(entries) == 0 {
		return "no matches found", nil
	}

	result := applyEntryPagination(entries, mode, opts.Offset, opts.HeadLimit)

	if matchCount >= maxMatches {
		result += fmt.Sprintf("\n... [truncated: %d matches shown, limit reached]\n", maxMatches)
	}
	return result, nil
}

// matchesGrepFilter checks if a file path passes glob and type filters.
func matchesGrepFilter(fpath string, opts grepOptions) bool {
	if opts.Glob != "" {
		if !matchGlobPath(fpath, opts.Glob) {
			return false
		}
	}
	if opts.FileType != "" {
		ext := strings.TrimPrefix(filepath.Ext(fpath), ".")
		if !matchFileType(ext, opts.FileType) {
			return false
		}
	}
	return true
}

// matchGlobPath matches a file path against a glob pattern.
// Supports ** patterns by matching against the full relative path.
func matchGlobPath(fpath, pattern string) bool {
	if strings.Contains(pattern, "**") {
		return matchDoubleStarGlob(fpath, pattern)
	}
	if strings.Contains(pattern, "/") {
		matched, err := filepath.Match(pattern, fpath)
		return err == nil && matched
	}
	matched, err := filepath.Match(pattern, filepath.Base(fpath))
	return err == nil && matched
}

// matchDoubleStarGlob handles ** glob patterns like "**/*.go" or "src/**/*.ts".
func matchDoubleStarGlob(fpath, pattern string) bool {
	parts := strings.SplitN(pattern, "**", 2)
	prefix := parts[0]
	suffix := ""
	if len(parts) > 1 {
		suffix = parts[1]
	}

	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		if !strings.HasPrefix(fpath, prefix) {
			return false
		}
	}

	if suffix != "" {
		suffix = strings.TrimPrefix(suffix, "/")
		if strings.Contains(suffix, "/") {
			return strings.HasSuffix(fpath, suffix)
		}
		matched, err := filepath.Match(suffix, filepath.Base(fpath))
		return err == nil && matched
	}

	return true
}

// fileTypeMap maps rg type names to file extensions.
var fileTypeMap = map[string][]string{
	"go":     {"go"},
	"py":     {"py"},
	"js":     {"js", "jsx", "mjs"},
	"ts":     {"ts", "tsx"},
	"rust":   {"rs"},
	"java":   {"java"},
	"rb":     {"rb"},
	"c":      {"c", "h"},
	"cpp":    {"cpp", "cc", "cxx", "hpp", "hxx", "h"},
	"css":    {"css"},
	"html":   {"html", "htm"},
	"json":   {"json"},
	"yaml":   {"yaml", "yml"},
	"toml":   {"toml"},
	"md":     {"md", "markdown"},
	"sh":     {"sh", "bash"},
	"sql":    {"sql"},
	"xml":    {"xml"},
	"tf":     {"tf"},
	"swift":  {"swift"},
	"kotlin": {"kt", "kts"},
}

// matchFileType checks if a file extension matches the specified type.
func matchFileType(ext, fileType string) bool {
	exts, ok := fileTypeMap[fileType]
	if !ok {
		return ext == fileType
	}
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

// fileHasMatch scans a file and returns true if any line matches.
func fileHasMatch(re *regexp.Regexp, path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if re.MatchString(scanner.Text()) {
			return true, nil
		}
	}
	return false, scanner.Err()
}

// countFileMatches counts the number of matching lines in a file.
func countFileMatches(re *regexp.Regexp, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if re.MatchString(scanner.Text()) {
			count++
		}
	}
	return count, scanner.Err()
}

// grepFileContent returns matching lines with optional context for a single file.
func grepFileContent(re *regexp.Regexp, path string, opts grepOptions) ([]string, error) {
	before := opts.effectiveBefore()
	after := opts.effectiveAfter()

	if before > 0 || after > 0 {
		return grepFileWithContext(re, path, opts, before, after)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			if opts.LineNumbers {
				entries = append(entries, fmt.Sprintf("%s:%d:%s", path, lineNum, line))
			} else {
				entries = append(entries, fmt.Sprintf("%s:%s", path, line))
			}
		}
	}
	return entries, scanner.Err()
}

// grepFileWithContext returns matching lines with before/after context lines.
// Groups of overlapping context are merged and separated by "--".
func grepFileWithContext(re *regexp.Regexp, path string, opts grepOptions, before, after int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	// Remove trailing empty line from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Find all matching line indices
	var matchIndices []int
	for i, line := range lines {
		if re.MatchString(line) {
			matchIndices = append(matchIndices, i)
		}
	}

	if len(matchIndices) == 0 {
		return nil, nil
	}

	// Build ranges [start, end] for each match with context
	type lineRange struct{ start, end int }
	ranges := make([]lineRange, 0, len(matchIndices))
	for _, idx := range matchIndices {
		start := idx - before
		if start < 0 {
			start = 0
		}
		end := idx + after
		if end >= len(lines) {
			end = len(lines) - 1
		}
		ranges = append(ranges, lineRange{start, end})
	}

	// Merge overlapping ranges
	merged := []lineRange{ranges[0]}
	for i := 1; i < len(ranges); i++ {
		last := &merged[len(merged)-1]
		if ranges[i].start <= last.end+1 {
			if ranges[i].end > last.end {
				last.end = ranges[i].end
			}
		} else {
			merged = append(merged, ranges[i])
		}
	}

	// Build match set for highlighting
	matchSet := make(map[int]bool, len(matchIndices))
	for _, idx := range matchIndices {
		matchSet[idx] = true
	}

	// Emit groups separated by "--"
	var groups []string
	for _, r := range merged {
		var groupLines []string
		for i := r.start; i <= r.end; i++ {
			sep := "-"
			if matchSet[i] {
				sep = ":"
			}
			if opts.LineNumbers {
				groupLines = append(groupLines, fmt.Sprintf("%s%s%d%s%s", path, sep, i+1, sep, lines[i]))
			} else {
				groupLines = append(groupLines, fmt.Sprintf("%s%s%s", path, sep, lines[i]))
			}
		}
		groups = append(groups, strings.Join(groupLines, "\n"))
	}

	// Return as a single entry with groups separated by "--"
	return []string{strings.Join(groups, "\n--\n")}, nil
}

// grepMultiline reads entire files and matches across line boundaries.
func grepMultiline(re *regexp.Regexp, opts grepOptions, mode string) (string, error) {
	var entries []string
	matchCount := 0

	walkErr := filepath.WalkDir(opts.Path, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesGrepFilter(fpath, opts) {
			return nil
		}

		data, readErr := os.ReadFile(fpath)
		if readErr != nil {
			return nil
		}
		content := string(data)

		matches := re.FindAllStringIndex(content, -1)
		if len(matches) == 0 {
			return nil
		}

		switch mode {
		case "files_with_matches":
			entries = append(entries, fpath)
			matchCount++
		case "count":
			entries = append(entries, fmt.Sprintf("%s:%d", fpath, len(matches)))
			matchCount++
		default: // "content"
			for _, loc := range matches {
				matchText := content[loc[0]:loc[1]]
				startLine := strings.Count(content[:loc[0]], "\n") + 1
				if opts.LineNumbers {
					entries = append(entries, fmt.Sprintf("%s:%d:%s", fpath, startLine, matchText))
				} else {
					entries = append(entries, fmt.Sprintf("%s:%s", fpath, matchText))
				}
				matchCount++
			}
		}

		if matchCount >= maxMatches {
			return errMatchLimitReached
		}
		return nil
	})

	if walkErr != nil && walkErr != errMatchLimitReached {
		return "", fmt.Errorf("walking %s: %w", opts.Path, walkErr)
	}

	if len(entries) == 0 {
		return "no matches found", nil
	}

	return applyEntryPagination(entries, mode, opts.Offset, opts.HeadLimit), nil
}

// applyEntryPagination applies offset and head_limit to a slice of entries.
func applyEntryPagination(entries []string, mode string, offset, headLimit int) string {
	if offset > 0 {
		if offset >= len(entries) {
			return "no matches found"
		}
		entries = entries[offset:]
	}

	if headLimit > 0 && headLimit < len(entries) {
		entries = entries[:headLimit]
	}

	sep := "\n"
	if mode == "content" {
		// For content entries that may already contain context groups, use simple newline
		sep = "\n"
	}
	return strings.Join(entries, sep) + "\n"
}

// matchGlob checks if name matches the given glob pattern.
func matchGlob(name, pattern string) bool {
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

