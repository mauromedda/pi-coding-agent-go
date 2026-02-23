// ABOUTME: Tests for template variable rendering in prompt fragments
// ABOUTME: Validates substitution, missing vars, conditionals, and error handling

package prompts

import (
	"strings"
	"testing"
)

func TestRenderVariables_AllVars(t *testing.T) {
	t.Parallel()

	content := "Date: {{.DATE}}, CWD: {{.CWD}}"
	vars := map[string]string{
		"DATE": "2026-02-23",
		"CWD":  "/home/user",
	}

	got, err := RenderVariables(content, vars)
	if err != nil {
		t.Fatalf("RenderVariables() error = %v", err)
	}
	want := "Date: 2026-02-23, CWD: /home/user"
	if got != want {
		t.Errorf("RenderVariables() = %q; want %q", got, want)
	}
}

func TestRenderVariables_MissingVar(t *testing.T) {
	t.Parallel()

	content := "Date: {{.DATE}}, Tools: {{.TOOL_LIST}}"
	vars := map[string]string{
		"DATE": "2026-02-23",
	}

	got, err := RenderVariables(content, vars)
	if err != nil {
		t.Fatalf("RenderVariables() error = %v", err)
	}
	if !strings.Contains(got, "2026-02-23") {
		t.Errorf("expected DATE value in output; got %q", got)
	}
	// Missing TOOL_LIST should produce empty string
	if strings.Contains(got, "TOOL_LIST") {
		t.Errorf("expected TOOL_LIST placeholder to be resolved; got %q", got)
	}
}

func TestRenderVariables_NoVars(t *testing.T) {
	t.Parallel()

	content := "No variables here."
	got, err := RenderVariables(content, nil)
	if err != nil {
		t.Fatalf("RenderVariables() error = %v", err)
	}
	if got != content {
		t.Errorf("RenderVariables() = %q; want %q", got, content)
	}
}

func TestRenderVariables_InvalidTemplate(t *testing.T) {
	t.Parallel()

	content := "Bad template: {{.DATE"
	_, err := RenderVariables(content, map[string]string{"DATE": "x"})
	if err == nil {
		t.Fatal("RenderVariables() expected error for invalid template; got nil")
	}
}

func TestRenderVariables_ConditionalBlock(t *testing.T) {
	t.Parallel()

	content := "{{if .TOOL_LIST}}Tools: {{.TOOL_LIST}}{{end}}"

	tests := []struct {
		name string
		vars map[string]string
		want string
	}{
		{
			"with value",
			map[string]string{"TOOL_LIST": "read,write"},
			"Tools: read,write",
		},
		{
			"empty value",
			map[string]string{"TOOL_LIST": ""},
			"",
		},
		{
			"missing key",
			map[string]string{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RenderVariables(content, tt.vars)
			if err != nil {
				t.Fatalf("RenderVariables() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderVariables() = %q; want %q", got, tt.want)
			}
		})
	}
}
