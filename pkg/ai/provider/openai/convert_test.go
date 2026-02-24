// ABOUTME: Tests for OpenAI message conversion with image support
// ABOUTME: Verifies multimodal tool results use image_url with data URIs

package openai

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestConvertMessages_ToolResultWithImages(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		Messages: []ai.Message{
			{
				Role: ai.RoleUser,
				Content: []ai.Content{
					{
						Type:       ai.ContentToolResult,
						ID:         "t1",
						ResultText: "[Image: test.png]",
						Images: []ai.ImageContent{
							{MediaType: "image/png", Data: "aGVsbG8="},
						},
					},
				},
			},
		},
	}

	msgs := convertMessages(ctx)

	// Should produce a tool message with multimodal content
	var toolMsg *chatMessage
	for i := range msgs {
		if msgs[i].Role == "tool" {
			toolMsg = &msgs[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("expected a tool message")
	}

	// Content should be an array of parts
	parts, ok := toolMsg.Content.([]map[string]any)
	if !ok {
		t.Fatalf("expected content to be []map[string]any; got %T", toolMsg.Content)
	}
	// 1 text + 1 image_url = 2
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts; got %d", len(parts))
	}

	// Text part
	if parts[0]["type"] != "text" {
		t.Errorf("expected first part type 'text'; got %v", parts[0]["type"])
	}

	// Image part
	if parts[1]["type"] != "image_url" {
		t.Errorf("expected second part type 'image_url'; got %v", parts[1]["type"])
	}
	imgURL, ok := parts[1]["image_url"].(map[string]any)
	if !ok {
		t.Fatalf("expected image_url to be map; got %T", parts[1]["image_url"])
	}
	expectedURL := "data:image/png;base64,aGVsbG8="
	if imgURL["url"] != expectedURL {
		t.Errorf("expected url %q; got %v", expectedURL, imgURL["url"])
	}
}

func TestConvertMessages_ToolResultWithoutImages(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		Messages: []ai.Message{
			{
				Role: ai.RoleUser,
				Content: []ai.Content{
					{
						Type:       ai.ContentToolResult,
						ID:         "t1",
						ResultText: "file content",
					},
				},
			},
		},
	}

	msgs := convertMessages(ctx)

	var toolMsg *chatMessage
	for i := range msgs {
		if msgs[i].Role == "tool" {
			toolMsg = &msgs[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("expected a tool message")
	}

	// Content should be a plain string
	content, ok := toolMsg.Content.(string)
	if !ok {
		t.Fatalf("expected content to be string; got %T", toolMsg.Content)
	}
	if content != "file content" {
		t.Errorf("expected 'file content'; got %q", content)
	}
}
