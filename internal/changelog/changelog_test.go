// ABOUTME: Tests for the embedded changelog reader
// ABOUTME: Verifies Get() returns non-empty content from the embedded CHANGELOG.md

package changelog

import (
	"strings"
	"testing"
)

func TestGet_ReturnsNonEmpty(t *testing.T) {
	t.Parallel()

	content := Get()
	if content == "" {
		t.Fatal("Get() returned empty string; expected changelog content")
	}
}

func TestGet_ContainsChangelogHeader(t *testing.T) {
	t.Parallel()

	content := Get()
	if !strings.Contains(content, "# Changelog") {
		t.Errorf("expected changelog to contain '# Changelog' header, got:\n%s", content)
	}
}

func TestGet_ContainsUnreleasedSection(t *testing.T) {
	t.Parallel()

	content := Get()
	if !strings.Contains(content, "[Unreleased]") {
		t.Errorf("expected changelog to contain '[Unreleased]' section, got:\n%s", content)
	}
}
