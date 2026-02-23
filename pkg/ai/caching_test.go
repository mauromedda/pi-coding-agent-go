// ABOUTME: Tests for Anthropic prompt caching: cache_control application and JSON roundtrip
// ABOUTME: Verifies provider-gated caching on system prompt and tools

package ai

import (
	"encoding/json"
	"testing"
)

func TestApplyPromptCaching_Anthropic(t *testing.T) {
	ctx := &Context{
		System: "You are a helpful assistant.",
		Tools: []Tool{
			{Name: "read", Description: "Read a file"},
			{Name: "write", Description: "Write a file"},
		},
	}

	applied := ApplyPromptCaching(ctx, ApiAnthropic)

	if !applied {
		t.Error("expected ApplyPromptCaching to return true for Anthropic")
	}
	if ctx.SystemCacheControl == nil {
		t.Fatal("expected SystemCacheControl to be set")
	}
	if ctx.SystemCacheControl.Type != "ephemeral" {
		t.Errorf("SystemCacheControl.Type = %q, want \"ephemeral\"", ctx.SystemCacheControl.Type)
	}
	// Last tool should have cache_control
	lastTool := ctx.Tools[len(ctx.Tools)-1]
	if lastTool.CacheControl == nil {
		t.Fatal("expected last tool to have CacheControl set")
	}
	if lastTool.CacheControl.Type != "ephemeral" {
		t.Errorf("last tool CacheControl.Type = %q, want \"ephemeral\"", lastTool.CacheControl.Type)
	}
	// First tool should NOT have cache_control
	if ctx.Tools[0].CacheControl != nil {
		t.Error("expected first tool to NOT have CacheControl")
	}
}

func TestApplyPromptCaching_OpenAI(t *testing.T) {
	ctx := &Context{
		System: "You are a helpful assistant.",
		Tools:  []Tool{{Name: "read", Description: "Read a file"}},
	}

	applied := ApplyPromptCaching(ctx, ApiOpenAI)

	if applied {
		t.Error("expected ApplyPromptCaching to return false for OpenAI")
	}
	if ctx.SystemCacheControl != nil {
		t.Error("expected SystemCacheControl to be nil for OpenAI")
	}
	if ctx.Tools[0].CacheControl != nil {
		t.Error("expected tool CacheControl to be nil for OpenAI")
	}
}

func TestApplyPromptCaching_Google(t *testing.T) {
	ctx := &Context{
		System: "System prompt",
		Tools:  []Tool{{Name: "bash", Description: "Run commands"}},
	}

	applied := ApplyPromptCaching(ctx, ApiGoogle)

	if applied {
		t.Error("expected false for Google")
	}
	if ctx.SystemCacheControl != nil {
		t.Error("expected nil SystemCacheControl for Google")
	}
}

func TestApplyPromptCaching_AnthropicNoTools(t *testing.T) {
	ctx := &Context{
		System: "System prompt",
	}

	applied := ApplyPromptCaching(ctx, ApiAnthropic)

	if !applied {
		t.Error("expected true for Anthropic even with no tools")
	}
	if ctx.SystemCacheControl == nil {
		t.Fatal("expected SystemCacheControl to be set")
	}
}

func TestCacheControl_ContentJSON(t *testing.T) {
	c := Content{
		Type:         ContentText,
		Text:         "hello",
		CacheControl: &CacheControl{Type: "ephemeral"},
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Content
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.CacheControl == nil {
		t.Fatal("expected CacheControl after roundtrip")
	}
	if got.CacheControl.Type != "ephemeral" {
		t.Errorf("CacheControl.Type = %q, want \"ephemeral\"", got.CacheControl.Type)
	}
}

func TestCacheControl_ToolJSON(t *testing.T) {
	tool := Tool{
		Name:         "test",
		Description:  "A test tool",
		Parameters:   json.RawMessage(`{}`),
		CacheControl: &CacheControl{Type: "ephemeral"},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Tool
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.CacheControl == nil {
		t.Fatal("expected CacheControl after roundtrip")
	}
	if got.CacheControl.Type != "ephemeral" {
		t.Errorf("CacheControl.Type = %q, want \"ephemeral\"", got.CacheControl.Type)
	}
}

func TestCacheControl_OmittedWhenNil(t *testing.T) {
	c := Content{Type: ContentText, Text: "hello"}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify cache_control is not in the JSON when nil
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["cache_control"]; ok {
		t.Error("expected cache_control to be omitted from JSON when nil")
	}
}
