// ABOUTME: Tests for token estimation: text heuristic, content blocks, messages
// ABOUTME: Verifies chars÷4 accuracy, content type dispatch, aggregation

package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"short", "hello", 2},             // 5/4 = 1.25 → ceil = 2
		{"medium", "hello world!", 3},      // 12/4 = 3
		{"exactly divisible", "abcd", 1},   // 4/4 = 1
		{"one char", "a", 1},               // 1/4 → ceil = 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d; want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestEstimateContentTokens(t *testing.T) {
	tests := []struct {
		name    string
		content ai.Content
		wantMin int // minimum expected tokens (exact depends on overhead)
	}{
		{"text", ai.Content{Type: ai.ContentText, Text: "hello world"}, 1},
		{"thinking", ai.Content{Type: ai.ContentThinking, Thinking: "reasoning"}, 1},
		{"tool_use", ai.Content{Type: ai.ContentToolUse, Name: "Read", Input: json.RawMessage(`{"path":"/tmp"}`)}, 1},
		{"tool_result", ai.Content{Type: ai.ContentToolResult, ResultText: "file contents here"}, 1},
		{"image", ai.Content{Type: ai.ContentImage}, 1000},
		{"unknown", ai.Content{Type: "unknown"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateContentTokens(tt.content)
			if got < tt.wantMin {
				t.Errorf("EstimateContentTokens() = %d; want at least %d", got, tt.wantMin)
			}
		})
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	msg := ai.NewTextMessage(ai.RoleUser, "What is the weather?")
	tokens := EstimateMessageTokens(msg)

	// "What is the weather?" = 20 chars → 5 tokens + 4 overhead = 9
	if tokens != 9 {
		t.Errorf("EstimateMessageTokens = %d; want 9", tokens)
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "hi there"),
	}
	tokens := EstimateMessagesTokens(msgs)

	// "hello" = 5 chars → 2 + 4 = 6
	// "hi there" = 8 chars → 2 + 4 = 6
	// total = 12
	if tokens != 12 {
		t.Errorf("EstimateMessagesTokens = %d; want 12", tokens)
	}
}

func TestEstimateMessagesTokens_LargeConversation(t *testing.T) {
	// Generate a conversation with ~100K tokens worth of text
	largeText := strings.Repeat("word ", 10000) // 50000 chars → ~12500 tokens
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, largeText),
		ai.NewTextMessage(ai.RoleAssistant, largeText),
	}
	tokens := EstimateMessagesTokens(msgs)

	// Sanity check: should be roughly 25000 tokens + overhead
	if tokens < 20000 || tokens > 30000 {
		t.Errorf("EstimateMessagesTokens for large conversation = %d; expected ~25000", tokens)
	}
}
