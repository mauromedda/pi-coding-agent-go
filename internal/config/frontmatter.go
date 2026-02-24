// ABOUTME: Generic YAML frontmatter parser with CRLF normalization
// ABOUTME: Extracts typed frontmatter from Markdown content with --- delimiters

package config

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

// ParseFrontmatter extracts YAML frontmatter from Markdown content.
// It returns the parsed frontmatter as T, the remaining body, and any error.
// If no frontmatter is found, it returns (zero T, original content, nil).
// If the opening delimiter is present but the closing one is missing, it returns an error.
func ParseFrontmatter[T any](content string) (T, string, error) {
	var zero T

	// Normalize CRLF to LF.
	normalized := strings.ReplaceAll(content, "\r\n", "\n")

	// Check for opening delimiter.
	if !strings.HasPrefix(normalized, frontmatterDelimiter+"\n") {
		return zero, content, nil
	}

	// Find closing delimiter after the opening line.
	rest := normalized[len(frontmatterDelimiter)+1:]

	var yamlContent string
	var afterClosing string

	if strings.HasPrefix(rest, frontmatterDelimiter+"\n") || rest == frontmatterDelimiter {
		// Empty frontmatter: closing delimiter immediately follows opening.
		yamlContent = ""
		afterClosing = rest[len(frontmatterDelimiter):]
	} else {
		before, after, ok := strings.Cut(rest, "\n"+frontmatterDelimiter)
		if !ok {
			return zero, "", errors.New("unterminated frontmatter: missing closing ---")
		}
		yamlContent = before
		afterClosing = after
	}

	body := strings.TrimPrefix(afterClosing, "\n")

	var result T
	if err := yaml.Unmarshal([]byte(yamlContent), &result); err != nil {
		return zero, "", fmt.Errorf("parse frontmatter YAML: %w", err)
	}

	return result, body, nil
}

// StripFrontmatter returns the body portion of content after removing any
// YAML frontmatter. If no frontmatter is present, the original content is returned.
func StripFrontmatter(content string) string {
	_, body, err := ParseFrontmatter[map[string]any](content)
	if err != nil {
		return content
	}
	return body
}
