// ABOUTME: Tests for YAML frontmatter parser: basic, CRLF, missing, unterminated
// ABOUTME: Verifies generic type parsing, multiline YAML values, empty frontmatter

package config

import (
	"testing"
)

type testFM struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"allowed-tools"`
}

func TestParseFrontmatter_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantBody string
	}{
		{
			name:     "standard frontmatter",
			input:    "---\nname: test\n---\nbody content",
			wantName: "test",
			wantBody: "body content",
		},
		{
			name:     "frontmatter with description",
			input:    "---\nname: myskill\ndescription: A useful skill\n---\nremaining text",
			wantName: "myskill",
			wantBody: "remaining text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, body, err := ParseFrontmatter[testFM](tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if body != tt.wantBody {
				t.Errorf("Body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestParseFrontmatter_CRLF(t *testing.T) {
	t.Parallel()

	input := "---\r\nname: test\r\ndescription: hello\r\n---\r\nbody here"

	got, body, err := ParseFrontmatter[testFM](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
	if got.Description != "hello" {
		t.Errorf("Description = %q, want %q", got.Description, "hello")
	}
	if body != "body here" {
		t.Errorf("Body = %q, want %q", body, "body here")
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "plain markdown",
			input: "# Hello World\n\nSome content here.",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "starts with dashes but not exactly three",
			input: "----\nname: test\n----\nbody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, body, err := ParseFrontmatter[testFM](tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != "" || got.Description != "" || len(got.Tools) != 0 {
				t.Errorf("expected zero value, got %+v", got)
			}
			if body != tt.input {
				t.Errorf("Body = %q, want %q", body, tt.input)
			}
		})
	}
}

func TestParseFrontmatter_Unterminated(t *testing.T) {
	t.Parallel()

	input := "---\nname: test\nbody without closing"

	_, _, err := ParseFrontmatter[testFM](input)
	if err == nil {
		t.Fatal("expected error for unterminated frontmatter, got nil")
	}
}

func TestParseFrontmatter_MultilineYAML(t *testing.T) {
	t.Parallel()

	input := "---\nname: multi\nallowed-tools:\n  - Read\n  - Write\n  - Edit\n---\nbody"

	got, body, err := ParseFrontmatter[testFM](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "multi" {
		t.Errorf("Name = %q, want %q", got.Name, "multi")
	}
	wantTools := []string{"Read", "Write", "Edit"}
	if len(got.Tools) != len(wantTools) {
		t.Fatalf("Tools length = %d, want %d", len(got.Tools), len(wantTools))
	}
	for i, tool := range got.Tools {
		if tool != wantTools[i] {
			t.Errorf("Tools[%d] = %q, want %q", i, tool, wantTools[i])
		}
	}
	if body != "body" {
		t.Errorf("Body = %q, want %q", body, "body")
	}
}

func TestParseFrontmatter_Empty(t *testing.T) {
	t.Parallel()

	input := "---\n---\nbody here"

	got, body, err := ParseFrontmatter[testFM](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "" || got.Description != "" || len(got.Tools) != 0 {
		t.Errorf("expected zero value, got %+v", got)
	}
	if body != "body here" {
		t.Errorf("Body = %q, want %q", body, "body here")
	}
}

func TestParseFrontmatter_WithColonsInValues(t *testing.T) {
	t.Parallel()

	input := "---\nname: myapp\ndescription: \"localhost:8080/api\"\n---\nbody"

	got, body, err := ParseFrontmatter[testFM](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "myapp" {
		t.Errorf("Name = %q, want %q", got.Name, "myapp")
	}
	if got.Description != "localhost:8080/api" {
		t.Errorf("Description = %q, want %q", got.Description, "localhost:8080/api")
	}
	if body != "body" {
		t.Errorf("Body = %q, want %q", body, "body")
	}
}

func TestStripFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantBody string
	}{
		{
			name:     "with frontmatter",
			input:    "---\nname: test\n---\nbody only",
			wantBody: "body only",
		},
		{
			name:     "without frontmatter",
			input:    "just plain text",
			wantBody: "just plain text",
		},
		{
			name:     "empty input",
			input:    "",
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := StripFrontmatter(tt.input)
			if got != tt.wantBody {
				t.Errorf("StripFrontmatter() = %q, want %q", got, tt.wantBody)
			}
		})
	}
}
