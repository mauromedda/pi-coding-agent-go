// ABOUTME: Parse @file#line-line syntax from user input
// ABOUTME: Resolves file paths relative to workDir and extracts line ranges

package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FileMention represents a parsed @file#line-line reference.
type FileMention struct {
	Path      string
	StartLine int // 0 if not specified
	EndLine   int // 0 if not specified
}

var mentionRegex = regexp.MustCompile(`@([\w./_-]+(?:#\d+(?:-\d+)?)?)`)

// ParseMentions extracts all @file#line-line references from input text.
// Returns the cleaned text and parsed mentions.
func ParseMentions(input, workDir string) (string, []FileMention, error) {
	matches := mentionRegex.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input, nil, nil
	}

	var mentions []FileMention
	cleaned := input

	// Process in reverse order to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		fullStart := matches[i][0]
		fullEnd := matches[i][1]
		ref := input[matches[i][2]:matches[i][3]]

		mention, err := parseRef(ref, workDir)
		if err != nil {
			continue // Skip invalid mentions
		}

		mentions = append([]FileMention{mention}, mentions...)

		// Build replacement content
		content, err := readMentionContent(mention)
		if err != nil {
			continue
		}

		replacement := fmt.Sprintf("\n[File: %s", mention.Path)
		if mention.StartLine > 0 {
			replacement += fmt.Sprintf("#%d-%d", mention.StartLine, mention.EndLine)
		}
		replacement += "]\n```\n" + content + "\n```\n"

		cleaned = cleaned[:fullStart] + replacement + cleaned[fullEnd:]
	}

	return cleaned, mentions, nil
}

func parseRef(ref, workDir string) (FileMention, error) {
	parts := strings.SplitN(ref, "#", 2)
	path := parts[0]

	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	mention := FileMention{Path: path}

	if len(parts) == 2 {
		lineRange := parts[1]
		rangeParts := strings.SplitN(lineRange, "-", 2)

		start, err := strconv.Atoi(rangeParts[0])
		if err != nil {
			return mention, fmt.Errorf("invalid line number %q: %w", rangeParts[0], err)
		}
		mention.StartLine = start

		if len(rangeParts) == 2 {
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return mention, fmt.Errorf("invalid end line %q: %w", rangeParts[1], err)
			}
			mention.EndLine = end
		} else {
			mention.EndLine = start
		}
	}

	return mention, nil
}

func readMentionContent(mention FileMention) (string, error) {
	data, err := os.ReadFile(mention.Path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", mention.Path, err)
	}

	if mention.StartLine == 0 {
		return string(data), nil
	}

	lines := strings.Split(string(data), "\n")
	start := mention.StartLine - 1
	end := mention.EndLine
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= len(lines) {
		return "", nil
	}

	return strings.Join(lines[start:end], "\n"), nil
}
