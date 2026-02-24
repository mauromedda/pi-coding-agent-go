// ABOUTME: Dedicated image reading tool for LLM vision analysis
// ABOUTME: Validates image extension, size, and sandbox; reuses handleImageFile from read.go

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
)

// newReadImageTool creates a read-only tool dedicated to reading image files.
// Unlike the general `read` tool, this tool rejects non-image files and
// its description guides LLMs to use it for visual analysis tasks.
func newReadImageTool(sb *permission.Sandbox) *agent.AgentTool {
	return &agent.AgentTool{
		Name:  "read_image",
		Label: "Read Image",
		Description: "Read an image file and return its contents for visual analysis. " +
			"Use this tool when you need to see or describe an image. " +
			"Supports PNG, JPEG, GIF, and WebP formats.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path"],
			"properties": {
				"path": {"type": "string", "description": "Absolute path to the image file"}
			}
		}`),
		ReadOnly: true,
		Execute: func(ctx context.Context, id string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
			return executeReadImage(sb, params)
		},
	}
}

func executeReadImage(sb *permission.Sandbox, params map[string]any) (agent.ToolResult, error) {
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

	mime, ok := imageExtMIME(path)
	if !ok {
		return errResult(fmt.Errorf("not a supported image format: %s (supported: png, jpg, jpeg, gif, webp)", path)), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return errResult(fmt.Errorf("opening image %s: %w", path, err)), nil
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxImageFileSize+1))
	if err != nil {
		return errResult(fmt.Errorf("reading image %s: %w", path, err)), nil
	}

	return handleImageFile(data, path, mime), nil
}
