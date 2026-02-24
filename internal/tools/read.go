// ABOUTME: Read-file tool: returns file contents with optional offset/limit
// ABOUTME: Detects binary files, truncates large outputs, validates paths via sandbox

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/types"
)

const (
	maxReadOutput    = 100 * 1024       // 100KB
	maxFileReadSize  = 10 * 1024 * 1024 // 10MB: cap on bytes read from disk
	binaryCheckBytes = 512
)

// NewReadTool creates a read-only tool that returns file contents.
func NewReadTool() *agent.AgentTool {
	return newReadTool(nil)
}

// NewReadToolWithSandbox creates a read tool that validates paths against the sandbox.
func NewReadToolWithSandbox(sb *permission.Sandbox) *agent.AgentTool {
	return newReadTool(sb)
}

func newReadTool(sb *permission.Sandbox) *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "read",
		Label:       "Read File",
		Description: "Read the contents of a file. Supports optional offset and limit in lines.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path"],
			"properties": {
				"path":   {"type": "string", "description": "Absolute path to the file"},
				"offset": {"type": "integer", "description": "Line number to start reading from (0-based)"},
				"limit":  {"type": "integer", "description": "Maximum number of lines to return"}
			}
		}`),
		ReadOnly: true,
		Execute: func(ctx context.Context, id string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
			return executeRead(sb, ctx, id, params, onUpdate)
		},
	}
}

func executeRead(sb *permission.Sandbox, _ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	rawPath, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	cwd, _ := os.Getwd()
	path := ResolveReadPath(rawPath, cwd)

	if sb != nil {
		if err := sb.ValidatePath(path); err != nil {
			return errResult(err), nil
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return errResult(fmt.Errorf("reading file %s: %w", path, err)), nil
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxFileReadSize))
	if err != nil {
		return errResult(fmt.Errorf("reading file %s: %w", path, err)), nil
	}

	if isBinary(data) {
		if mime, ok := imageExtMIME(path); ok {
			return handleImageFile(data, path, mime), nil
		}
		return agent.ToolResult{Content: fmt.Sprintf("binary file detected: %s", path), IsError: true}, nil
	}

	content := applyOffsetLimit(string(data), params)
	content = truncateOutput(content, maxReadOutput)

	return agent.ToolResult{Content: content}, nil
}

// isBinary checks for null bytes in the first binaryCheckBytes of data.
func isBinary(data []byte) bool {
	limit := min(len(data), binaryCheckBytes)
	return slices.Contains(data[:limit], 0)
}

// applyOffsetLimit extracts a line range from content based on offset/limit params.
func applyOffsetLimit(content string, params map[string]any) string {
	lines := splitLines(content)

	offset := intParam(params, "offset", 0)
	limit := intParam(params, "limit", 0)

	if offset > len(lines) {
		offset = len(lines)
	}
	lines = lines[offset:]

	if limit > 0 && limit < len(lines) {
		lines = lines[:limit]
	}

	return joinLines(lines)
}

// splitLines splits content into lines, preserving trailing newlines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.SplitAfter(s, "\n")
}

// joinLines concatenates lines back into a single string using strings.Builder.
func joinLines(lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
	}
	return b.String()
}

// truncateOutput limits output to maxBytes, appending a truncation notice.
func truncateOutput(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n... [output truncated]"
}

// imageExtensions maps file extensions to MIME types for supported image formats.
var imageExtensions = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// imageExtMIME returns the MIME type if the file extension is a supported image format.
func imageExtMIME(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	mime, ok := imageExtensions[ext]
	return mime, ok
}

const maxImageFileSize = 4_500_000 // 4.5 MB

// handleImageFile returns a ToolResult with image metadata and an ImageBlock.
// The Images field carries raw bytes used both for TUI rendering and for
// base64 injection into LLM context (when the model supports images).
func handleImageFile(data []byte, path, mime string) agent.ToolResult {
	if len(data) > maxImageFileSize {
		return agent.ToolResult{
			Content: fmt.Sprintf("image file too large: %s (%d bytes, max %d)", path, len(data), maxImageFileSize),
			IsError: true,
		}
	}
	content := fmt.Sprintf("[Image: %s %s (%d bytes)]", filepath.Base(path), mime, len(data))
	return agent.ToolResult{
		Content: content,
		Images: []types.ImageBlock{
			{
				Data:     data,
				MimeType: mime,
				Filename: filepath.Base(path),
			},
		},
	}
}
