// ABOUTME: Tests for Vertex AI message conversion with image and tool support
// ABOUTME: Verifies InlineData, FunctionCall, and FunctionResponse parts

package vertex

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBuildVertexRequestBody_ToolResultWithImages(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		Messages: []ai.Message{
			{
				Role: ai.RoleUser,
				Content: []ai.Content{
					{
						Type:       ai.ContentToolResult,
						Name:       "read_image",
						ResultText: "[Image: test.png]",
						Images: []ai.ImageContent{
							{MediaType: "image/png", Data: "aGVsbG8="},
						},
					},
				},
			},
		},
	}

	req := buildVertexRequestBody(ctx, nil)

	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content; got %d", len(req.Contents))
	}

	parts := req.Contents[0].Parts
	// FunctionResponse + InlineData = 2 parts
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts; got %d", len(parts))
	}

	if parts[0].FunctionResponse == nil {
		t.Fatal("expected FunctionResponse in first part")
	}
	if parts[1].InlineData == nil {
		t.Fatal("expected InlineData in second part")
	}
	if parts[1].InlineData.MimeType != "image/png" {
		t.Errorf("expected mimeType 'image/png'; got %q", parts[1].InlineData.MimeType)
	}
}

func TestBuildVertexRequestBody_ToolUse(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		Messages: []ai.Message{
			{
				Role: ai.RoleAssistant,
				Content: []ai.Content{
					{
						Type:  ai.ContentToolUse,
						Name:  "read",
						Input: []byte(`{"path":"/tmp"}`),
					},
				},
			},
		},
	}

	req := buildVertexRequestBody(ctx, nil)

	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content; got %d", len(req.Contents))
	}

	parts := req.Contents[0].Parts
	if len(parts) != 1 {
		t.Fatalf("expected 1 part; got %d", len(parts))
	}
	if parts[0].FunctionCall == nil {
		t.Fatal("expected FunctionCall part")
	}
	if parts[0].FunctionCall.Name != "read" {
		t.Errorf("expected name 'read'; got %q", parts[0].FunctionCall.Name)
	}
}

func TestBuildVertexRequestBody_ToolResultNoImages(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		Messages: []ai.Message{
			{
				Role: ai.RoleUser,
				Content: []ai.Content{
					{
						Type:       ai.ContentToolResult,
						Name:       "read",
						ResultText: "file content",
					},
				},
			},
		},
	}

	req := buildVertexRequestBody(ctx, nil)

	parts := req.Contents[0].Parts
	if len(parts) != 1 {
		t.Fatalf("expected 1 part; got %d", len(parts))
	}
	if parts[0].FunctionResponse == nil {
		t.Fatal("expected FunctionResponse")
	}
	if parts[0].InlineData != nil {
		t.Error("expected no InlineData")
	}
}
