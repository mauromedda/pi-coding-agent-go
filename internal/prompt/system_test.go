// ABOUTME: Tests for system prompt construction and output style instructions
// ABOUTME: Covers StyleInstructions mapping and BuildSystem style integration

package prompt

import (
	"strings"
	"testing"
)

func TestStyleInstructions_Concise(t *testing.T) {
	got := StyleInstructions("concise")
	if got == "" {
		t.Fatal("StyleInstructions(\"concise\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "concise") {
		t.Errorf("expected text containing \"concise\", got %q", got)
	}
}

func TestStyleInstructions_Verbose(t *testing.T) {
	got := StyleInstructions("verbose")
	if got == "" {
		t.Fatal("StyleInstructions(\"verbose\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "detailed") {
		t.Errorf("expected text containing \"detailed\", got %q", got)
	}
}

func TestStyleInstructions_Formal(t *testing.T) {
	got := StyleInstructions("formal")
	if got == "" {
		t.Fatal("StyleInstructions(\"formal\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "formal") {
		t.Errorf("expected text containing \"formal\", got %q", got)
	}
}

func TestStyleInstructions_Casual(t *testing.T) {
	got := StyleInstructions("casual")
	if got == "" {
		t.Fatal("StyleInstructions(\"casual\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "casual") {
		t.Errorf("expected text containing \"casual\", got %q", got)
	}
}

func TestStyleInstructions_Empty(t *testing.T) {
	got := StyleInstructions("")
	if got != "" {
		t.Errorf("StyleInstructions(\"\") = %q; want empty string", got)
	}
}

func TestStyleInstructions_Unknown(t *testing.T) {
	got := StyleInstructions("unknown-style")
	if got != "" {
		t.Errorf("StyleInstructions(\"unknown-style\") = %q; want empty string", got)
	}
}

func TestBuildSystem_WithStyle(t *testing.T) {
	opts := SystemOpts{
		CWD:   "/tmp/test",
		Style: "concise",
	}
	result := BuildSystem(opts)
	if !strings.Contains(strings.ToLower(result), "concise") {
		t.Errorf("BuildSystem with style \"concise\" should contain style text, got:\n%s", result)
	}
}

func TestBuildSystem_WithoutStyle(t *testing.T) {
	opts := SystemOpts{
		CWD: "/tmp/test",
	}
	result := BuildSystem(opts)

	// Should not contain any style instruction keywords
	styleKeywords := []string{
		"Be extremely concise",
		"detailed, thorough",
		"formal, professional",
		"casual and conversational",
	}
	for _, kw := range styleKeywords {
		if strings.Contains(result, kw) {
			t.Errorf("BuildSystem without style should not contain %q", kw)
		}
	}
}
