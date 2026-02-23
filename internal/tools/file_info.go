// ABOUTME: File info tool: returns metadata (size, lines, language, binary) without reading content
// ABOUTME: Uses os.Stat, bufio line counting, and extension-based language detection

package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewFileInfoTool creates a read-only tool that returns file metadata.
func NewFileInfoTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "file_info",
		Label:       "File Info",
		Description: "Get file metadata (size, lines, language, permissions) without reading the full content.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path"],
			"properties": {
				"path": {"type": "string", "description": "Path to the file or directory"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeFileInfo,
	}
}

func executeFileInfo(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	path, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return errResult(fmt.Errorf("stat %s: %w", path, err)), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "path: %s\n", path)
	fmt.Fprintf(&b, "size: %d bytes\n", info.Size())

	if info.IsDir() {
		fmt.Fprintf(&b, "type: directory\n")
		fmt.Fprintf(&b, "modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&b, "permissions: %s\n", info.Mode().String())
		return agent.ToolResult{Content: b.String()}, nil
	}

	fmt.Fprintf(&b, "type: file\n")

	lines, bin := countLinesAndBinary(path)
	fmt.Fprintf(&b, "lines: %d\n", lines)
	fmt.Fprintf(&b, "language: %s\n", extToLanguage(filepath.Ext(path)))
	fmt.Fprintf(&b, "modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "permissions: %s\n", info.Mode().String())
	fmt.Fprintf(&b, "binary: %v\n", bin)

	return agent.ToolResult{Content: b.String()}, nil
}

// countLinesAndBinary counts lines and checks for binary content in a single pass.
func countLinesAndBinary(path string) (int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()

	// Read first 512 bytes for binary detection.
	header := make([]byte, 512)
	n, _ := f.Read(header)
	bin := isBinary(header[:n])

	// Reset to count lines.
	f.Seek(0, 0)
	scanner := bufio.NewScanner(f)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, bin
}

// extToLanguage maps file extensions to language names.
func extToLanguage(ext string) string {
	m := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "javascript",
		".tsx":  "typescript",
		".rs":   "rust",
		".rb":   "ruby",
		".java": "java",
		".kt":   "kotlin",
		".c":    "c",
		".cpp":  "c++",
		".h":    "c",
		".hpp":  "c++",
		".cs":   "c#",
		".swift": "swift",
		".sh":   "shell",
		".md":   "markdown",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".toml": "toml",
		".sql":  "sql",
		".html": "html",
		".css":  "css",
	}
	if lang, ok := m[strings.ToLower(ext)]; ok {
		return lang
	}
	return "unknown"
}
