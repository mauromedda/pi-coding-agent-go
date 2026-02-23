// ABOUTME: Tests for diff rendering with colored unified diff output
// ABOUTME: Verifies color coding for added/removed lines and header detection

package btea

import (
	"strings"
	"testing"
)

func TestRenderDiff_ColoredOutput(t *testing.T) {
	diff := `--- a/main.go
+++ b/main.go
@@ -1,4 +1,4 @@
 package main

-func old() {}
+func new() {}
 // end`

	result := RenderDiff(diff)

	if result == "" {
		t.Fatal("RenderDiff returned empty string")
	}

	// Should contain the original lines (possibly with ANSI codes)
	if !strings.Contains(result, "func old()") {
		t.Error("result missing removed line")
	}
	if !strings.Contains(result, "func new()") {
		t.Error("result missing added line")
	}
}

func TestRenderDiff_EmptyInput(t *testing.T) {
	result := RenderDiff("")
	if result != "" {
		t.Errorf("RenderDiff(\"\") = %q; want empty", result)
	}
}

func TestRenderDiff_NoChanges(t *testing.T) {
	diff := `--- a/main.go
+++ b/main.go
@@ -1,2 +1,2 @@
 package main
 func hello() {}`

	result := RenderDiff(diff)
	if result == "" {
		t.Fatal("RenderDiff returned empty for context-only diff")
	}
}

func TestIsEditTool(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Edit", true},
		{"edit", true},
		{"Write", true},
		{"write", true},
		{"NotebookEdit", true},
		{"Read", false},
		{"Bash", false},
		{"Glob", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEditTool(tt.name); got != tt.want {
				t.Errorf("IsEditTool(%q) = %v; want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestComputeSimpleDiff(t *testing.T) {
	before := "line1\nline2\nline3\n"
	after := "line1\nmodified\nline3\n"

	diff := ComputeSimpleDiff(before, after)
	if diff == "" {
		t.Fatal("ComputeSimpleDiff returned empty string")
	}
	if !strings.Contains(diff, "-line2") {
		t.Error("diff missing removed line")
	}
	if !strings.Contains(diff, "+modified") {
		t.Error("diff missing added line")
	}
}

func TestComputeSimpleDiff_Identical(t *testing.T) {
	text := "same\ncontent\n"
	diff := ComputeSimpleDiff(text, text)
	if diff != "" {
		t.Errorf("identical content should return empty diff; got %q", diff)
	}
}
