// ABOUTME: Grep tool: searches file contents using ripgrep or built-in fallback
// ABOUTME: Supports output modes, context lines, case-insensitive, multiline, head_limit/offset

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// grepOptions carries parsed parameters for both rg and builtin grep paths.
type grepOptions struct {
	Pattern     string
	Path        string
	Glob        string // from "glob" or "include" (backward compat)
	FileType    string // rg --type (e.g., "go", "py")
	OutputMode  string // "content" | "files_with_matches" | "count"; default "files_with_matches"
	After       int    // -A context lines after match
	Before      int    // -B context lines before match
	Context     int    // -C context lines (overrides A/B when > 0)
	Insensitive bool   // -i case-insensitive
	LineNumbers bool   // -n show line numbers, default true
	HeadLimit   int    // 0 = unlimited
	Offset      int    // 0 = no skip
	Multiline   bool   // -U --multiline-dotall
}

// effectiveOutputMode returns the output mode, defaulting to "files_with_matches".
func (o grepOptions) effectiveOutputMode() string {
	if o.OutputMode == "" {
		return "files_with_matches"
	}
	return o.OutputMode
}

// effectiveBefore returns the before-context lines, considering Context override.
func (o grepOptions) effectiveBefore() int {
	if o.Context > 0 {
		return o.Context
	}
	return o.Before
}

// effectiveAfter returns the after-context lines, considering Context override.
func (o grepOptions) effectiveAfter() int {
	if o.Context > 0 {
		return o.Context
	}
	return o.After
}

// NewGrepTool creates a read-only tool that searches file contents.
func NewGrepTool(hasRg bool) *agent.AgentTool {
	return &agent.AgentTool{
		Name:  "grep",
		Label: "Search File Contents",
		Description: `A powerful search tool built on ripgrep.

Usage:
- Supports full regex syntax (e.g., "log.*Error", "function\s+\w+")
- Filter files with glob parameter (e.g., "*.js", "**/*.tsx") or type parameter (e.g., "js", "py", "rust")
- Output modes: "content" shows matching lines, "files_with_matches" shows only file paths (default), "count" shows match counts
- Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping
- Multiline matching: By default patterns match within single lines only. For cross-line patterns, use multiline: true

Output modes:
- "files_with_matches" (default): One file path per line; most efficient for finding which files contain a pattern
- "content": Shows matching lines with optional line numbers (-n, default true) and context (-A/-B/-C)
- "count": Shows path:count for each file with matches

Filtering:
- glob: Glob pattern to filter files (e.g., "*.js", "*.{ts,tsx}") - maps to rg --glob
- type: File type filter (e.g., "go", "py", "js") - maps to rg --type

Context lines (content mode only):
- -A: Lines after each match
- -B: Lines before each match
- -C/context: Lines before and after (overrides -A/-B)

Pagination:
- head_limit: Limit output to first N entries (0 = unlimited)
- offset: Skip first N entries before applying head_limit`,
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["pattern"],
			"properties": {
				"pattern":     {"type": "string", "description": "The regular expression pattern to search for in file contents"},
				"path":        {"type": "string", "description": "File or directory to search in. Defaults to current working directory."},
				"glob":        {"type": "string", "description": "Glob pattern to filter files (e.g. \"*.js\", \"*.{ts,tsx}\")"},
				"include":     {"type": "string", "description": "Alias for glob (backward compatibility)"},
				"type":        {"type": "string", "description": "File type to search (e.g., \"go\", \"py\", \"js\")"},
				"output_mode": {"type": "string", "enum": ["content", "files_with_matches", "count"], "description": "Output mode: content, files_with_matches (default), or count"},
				"-A":          {"type": "integer", "description": "Number of lines to show after each match (content mode only)"},
				"-B":          {"type": "integer", "description": "Number of lines to show before each match (content mode only)"},
				"-C":          {"type": "integer", "description": "Alias for context"},
				"context":     {"type": "integer", "description": "Number of lines before and after each match (overrides -A/-B)"},
				"-i":          {"type": "boolean", "description": "Case insensitive search"},
				"-n":          {"type": "boolean", "description": "Show line numbers (content mode, default true)"},
				"head_limit":  {"type": "integer", "description": "Limit output to first N entries (0 = unlimited)"},
				"offset":      {"type": "integer", "description": "Skip first N entries before applying head_limit (0 = no skip)"},
				"multiline":   {"type": "boolean", "description": "Enable multiline mode where . matches newlines"}
			}
		}`),
		ReadOnly: true,
		Execute:  makeGrepExecutor(hasRg),
	}
}

// parseGrepOptions extracts grepOptions from the parameter map.
func parseGrepOptions(params map[string]any) (grepOptions, error) {
	pattern, err := requireStringParam(params, "pattern")
	if err != nil {
		return grepOptions{}, err
	}

	glob := stringParam(params, "glob", "")
	if glob == "" {
		glob = stringParam(params, "include", "")
	}

	// -C and context are aliases; prefer context, fall back to -C
	ctx := intParam(params, "context", 0)
	if ctx == 0 {
		ctx = intParam(params, "-C", 0)
	}

	return grepOptions{
		Pattern:     pattern,
		Path:        stringParam(params, "path", "."),
		Glob:        glob,
		FileType:    stringParam(params, "type", ""),
		OutputMode:  stringParam(params, "output_mode", ""),
		After:       intParam(params, "-A", 0),
		Before:      intParam(params, "-B", 0),
		Context:     ctx,
		Insensitive: boolParam(params, "-i", false),
		LineNumbers: boolParam(params, "-n", true),
		HeadLimit:   intParam(params, "head_limit", 0),
		Offset:      intParam(params, "offset", 0),
		Multiline:   boolParam(params, "multiline", false),
	}, nil
}

func makeGrepExecutor(hasRg bool) func(context.Context, string, map[string]any, func(agent.ToolUpdate)) (agent.ToolResult, error) {
	return func(ctx context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
		opts, err := parseGrepOptions(params)
		if err != nil {
			return errResult(err), nil
		}

		var output string
		if hasRg {
			output, err = grepWithRg(ctx, opts)
		} else {
			output, err = grepBuiltin(opts)
		}
		if err != nil {
			return errResult(fmt.Errorf("grep: %w", err)), nil
		}

		output = truncateOutput(output, maxReadOutput)
		return agent.ToolResult{Content: output}, nil
	}
}

// grepWithRg runs ripgrep with the specified options.
func grepWithRg(ctx context.Context, opts grepOptions) (string, error) {
	mode := opts.effectiveOutputMode()
	args := buildRgArgs(opts, mode)
	args = append(args, opts.Path)

	cmd := exec.CommandContext(ctx, "rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "no matches found", nil
		}
		return "", fmt.Errorf("ripgrep failed: %s: %w", stderr.String(), err)
	}

	raw := stdout.String()
	return applyPagination(raw, mode, opts.Offset, opts.HeadLimit), nil
}

// buildRgArgs constructs ripgrep CLI arguments from grepOptions.
func buildRgArgs(opts grepOptions, mode string) []string {
	var args []string

	switch mode {
	case "files_with_matches":
		args = append(args, "-l")
	case "count":
		args = append(args, "-c")
	default: // "content"
		if opts.LineNumbers {
			args = append(args, "-n")
		}
		if b := opts.effectiveBefore(); b > 0 {
			args = append(args, fmt.Sprintf("-B%d", b))
		}
		if a := opts.effectiveAfter(); a > 0 {
			args = append(args, fmt.Sprintf("-A%d", a))
		}
	}

	if opts.Insensitive {
		args = append(args, "--ignore-case")
	}
	if opts.Multiline {
		args = append(args, "-U", "--multiline-dotall")
	}
	if opts.Glob != "" {
		args = append(args, "--glob", opts.Glob)
	}
	if opts.FileType != "" {
		args = append(args, "--type", opts.FileType)
	}

	args = append(args, "-e", opts.Pattern)
	return args
}

// applyPagination splits output into entries, applies offset and head_limit, and rejoins.
// For content mode with context, entries are separated by "--\n".
// For files_with_matches and count, each line is an entry.
func applyPagination(raw, mode string, offset, headLimit int) string {
	if offset == 0 && headLimit == 0 {
		return raw
	}

	raw = strings.TrimRight(raw, "\n")
	if raw == "" {
		return "no matches found"
	}

	var entries []string
	if mode == "content" {
		entries = strings.Split(raw, "\n--\n")
	} else {
		entries = strings.Split(raw, "\n")
	}

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
		sep = "\n--\n"
	}
	return strings.Join(entries, sep) + "\n"
}
