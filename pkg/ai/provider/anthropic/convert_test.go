// ABOUTME: Tests for Anthropic message conversion, especially image-bearing tool results
// ABOUTME: Verifies array-form content emission and backwards-compatible string form

package anthropic

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestConvertContentBlock_ToolResultWithImages(t *testing.T) {
	t.Parallel()

	b := ai.Content{
		Type:       ai.ContentToolResult,
		ID:         "t1",
		ResultText: "[Image: test.png]",
		Images: []ai.ImageContent{
			{MediaType: "image/png", Data: "aGVsbG8="},
		},
	}

	result := convertContentBlock(b)

	contentArr, ok := result["content"].([]map[string]any)
	if !ok {
		t.Fatalf("expected content to be []map[string]any; got %T", result["content"])
	}
	if len(contentArr) != 2 {
		t.Fatalf("expected 2 content parts; got %d", len(contentArr))
	}

	// First part: text
	if contentArr[0]["type"] != "text" {
		t.Errorf("expected first part type 'text'; got %v", contentArr[0]["type"])
	}
	if contentArr[0]["text"] != "[Image: test.png]" {
		t.Errorf("expected text '[Image: test.png]'; got %v", contentArr[0]["text"])
	}

	// Second part: image
	if contentArr[1]["type"] != "image" {
		t.Errorf("expected second part type 'image'; got %v", contentArr[1]["type"])
	}
	source, ok := contentArr[1]["source"].(map[string]any)
	if !ok {
		t.Fatalf("expected source to be map; got %T", contentArr[1]["source"])
	}
	if source["type"] != "base64" {
		t.Errorf("expected source type 'base64'; got %v", source["type"])
	}
	if source["media_type"] != "image/png" {
		t.Errorf("expected media_type 'image/png'; got %v", source["media_type"])
	}
	if source["data"] != "aGVsbG8=" {
		t.Errorf("expected data 'aGVsbG8='; got %v", source["data"])
	}
}

func TestConvertContentBlock_ToolResultWithoutImages(t *testing.T) {
	t.Parallel()

	b := ai.Content{
		Type:       ai.ContentToolResult,
		ID:         "t1",
		ResultText: "file content here",
	}

	result := convertContentBlock(b)

	// When no images, content should be a plain string (backwards compatible)
	content, ok := result["content"].(string)
	if !ok {
		t.Fatalf("expected content to be string; got %T", result["content"])
	}
	if content != "file content here" {
		t.Errorf("expected 'file content here'; got %q", content)
	}
}

func TestConvertContentBlock_ToolResultMultipleImages(t *testing.T) {
	t.Parallel()

	b := ai.Content{
		Type:       ai.ContentToolResult,
		ID:         "t1",
		ResultText: "two images",
		Images: []ai.ImageContent{
			{MediaType: "image/png", Data: "cG5n"},
			{MediaType: "image/jpeg", Data: "anBn"},
		},
	}

	result := convertContentBlock(b)

	contentArr, ok := result["content"].([]map[string]any)
	if !ok {
		t.Fatalf("expected content to be []map[string]any; got %T", result["content"])
	}
	// 1 text + 2 images = 3
	if len(contentArr) != 3 {
		t.Fatalf("expected 3 content parts; got %d", len(contentArr))
	}
}
