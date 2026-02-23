// ABOUTME: Find references tool: searches for symbol usage across files
// ABOUTME: Uses ripgrep when available, falls back to stdlib regex walk

package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

const maxRefResults = 200

// NewFindReferencesTool creates a read-only tool that finds symbol references.
func NewFindReferencesTool(hasRg bool) *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "find_references",
		Label:       "Find References",
		Description: "Search for references to a symbol across files. Uses ripgrep if available, stdlib fallback otherwise.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["symbol"],
			"properties": {
				"symbol":  {"type": "string", "description": "Symbol name to search for (word-boundary match)"},
				"path":    {"type": "string", "description": "Directory to search (default: current dir)"},
				"include": {"type": "string", "description": "Glob pattern to filter files (e.g. *.go)"}
			}
		}`),
		ReadOnly: true,
		Execute:  makeFindReferencesExecutor(hasRg),
	}
}

func makeFindReferencesExecutor(hasRg bool) func(context.Context, string, map[string]any, func(agent.ToolUpdate)) (agent.ToolResult, error) {
	return func(ctx context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
		symbol, err := requireStringParam(params, "symbol")
		if err != nil {
			return errResult(err), nil
		}

		path := stringParam(params, "path", ".")
		include := stringParam(params, "include", "")

		var output string
		if hasRg {
			output, err = refsWithRg(ctx, symbol, path, include)
		} else {
			output, err = refsBuiltin(symbol, path, include)
		}
		if err != nil {
			return errResult(fmt.Errorf("find_references: %w", err)), nil
		}

		if output == "" {
			return agent.ToolResult{Content: "no references found"}, nil
		}
		output = truncateOutput(output, maxReadOutput)
		return agent.ToolResult{Content: output}, nil
	}
}

func refsWithRg(ctx context.Context, symbol, path, include string) (string, error) {
	args := []string{"-n", "-w", symbol}
	if include != "" {
		args = append(args, "--glob", include)
	}
	args = append(args, path)

	cmd := exec.CommandContext(ctx, "rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("ripgrep: %s: %w", stderr.String(), err)
	}
	return stdout.String(), nil
}

func refsBuiltin(symbol, root, include string) (string, error) {
	pattern, err := regexp.Compile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	if err != nil {
		return "", fmt.Errorf("invalid symbol pattern: %w", err)
	}

	var includeGlob string
	if include != "" {
		includeGlob = include
	}

	var b strings.Builder
	count := 0

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if count >= maxRefResults {
			return fs.SkipAll
		}

		if includeGlob != "" {
			matched, _ := filepath.Match(includeGlob, d.Name())
			if !matched {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if pattern.MatchString(line) {
				rel, _ := filepath.Rel(root, path)
				if rel == "" {
					rel = path
				}
				fmt.Fprintf(&b, "%s:%d:%s\n", rel, lineNum, line)
				count++
				if count >= maxRefResults {
					return fs.SkipAll
				}
			}
		}
		return nil
	})

	if walkErr != nil {
		return "", walkErr
	}
	return b.String(), nil
}
