// ABOUTME: Tests for Google Gemini message conversion with image support
// ABOUTME: Verifies InlineData parts are emitted for tool results with images

package google

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBuildGeminiRequestBody_ToolResultWithImages(t *testing.T) {
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

	req := buildGeminiRequestBody(ctx, nil)

	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content; got %d", len(req.Contents))
	}

	parts := req.Contents[0].Parts
	// FunctionResponse + InlineData = 2 parts
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts; got %d", len(parts))
	}

	// First: FunctionResponse
	if parts[0].FunctionResponse == nil {
		t.Fatal("expected FunctionResponse in first part")
	}

	// Second: InlineData
	if parts[1].InlineData == nil {
		t.Fatal("expected InlineData in second part")
	}
	if parts[1].InlineData.MimeType != "image/png" {
		t.Errorf("expected mimeType 'image/png'; got %q", parts[1].InlineData.MimeType)
	}
	if parts[1].InlineData.Data != "aGVsbG8=" {
		t.Errorf("expected data 'aGVsbG8='; got %q", parts[1].InlineData.Data)
	}
}

func TestBuildGeminiRequestBody_ToolResultWithoutImages(t *testing.T) {
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

	req := buildGeminiRequestBody(ctx, nil)

	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content; got %d", len(req.Contents))
	}

	parts := req.Contents[0].Parts
	if len(parts) != 1 {
		t.Fatalf("expected 1 part; got %d", len(parts))
	}
	if parts[0].FunctionResponse == nil {
		t.Fatal("expected FunctionResponse")
	}
	if parts[0].InlineData != nil {
		t.Error("expected no InlineData for text-only result")
	}
}
