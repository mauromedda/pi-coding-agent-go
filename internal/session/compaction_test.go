// ABOUTME: Tests for compaction: file tracking, token-budget triggers, cut points, LLM compaction
// ABOUTME: Verifies CompactionEntry, ShouldCompact, FindCutPoint, CompactWithLLM

package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestExtractFileOps_ReadTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesRead) != 1 || entry.FilesRead[0] != "/src/main.go" {
		t.Errorf("expected FilesRead=[/src/main.go], got %v", entry.FilesRead)
	}
	if len(entry.FilesWritten) != 0 {
		t.Errorf("expected no FilesWritten, got %v", entry.FilesWritten)
	}
	if entry.MessageCount != 1 {
		t.Errorf("expected MessageCount=1, got %d", entry.MessageCount)
	}
}

func TestExtractFileOps_WriteTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Write",
					Input: json.RawMessage(`{"file_path":"/src/new.go","content":"package main"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesWritten) != 1 || entry.FilesWritten[0] != "/src/new.go" {
		t.Errorf("expected FilesWritten=[/src/new.go], got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_EditTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path":"/src/main.go","old_string":"foo","new_string":"bar"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesWritten) != 1 || entry.FilesWritten[0] != "/src/main.go" {
		t.Errorf("expected FilesWritten=[/src/main.go], got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_BashTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Bash",
					Input: json.RawMessage(`{"command":"ls -la"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	// Bash doesn't have file_path; should not track
	if len(entry.FilesRead) != 0 {
		t.Errorf("expected no FilesRead for Bash, got %v", entry.FilesRead)
	}
	if len(entry.FilesWritten) != 0 {
		t.Errorf("expected no FilesWritten for Bash, got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_MixedMessages(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "read main.go"),
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{Type: ai.ContentText, Text: "Let me read that file."},
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
			},
		},
		ai.NewTextMessage(ai.RoleUser, "now edit it"),
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path":"/src/main.go","old_string":"a","new_string":"b"}`),
				},
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/util.go"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if entry.MessageCount != 4 {
		t.Errorf("expected MessageCount=4, got %d", entry.MessageCount)
	}

	// main.go read + util.go read (deduplicated)
	if len(entry.FilesRead) != 2 {
		t.Errorf("expected 2 FilesRead, got %v", entry.FilesRead)
	}

	// main.go edited
	if len(entry.FilesWritten) != 1 {
		t.Errorf("expected 1 FilesWritten, got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_DeduplicatesFiles(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesRead) != 1 {
		t.Errorf("expected deduplicated FilesRead to have 1 entry, got %v", entry.FilesRead)
	}
}

func TestExtractFileOps_EmptyMessages(t *testing.T) {
	t.Parallel()

	entry := ExtractFileOps(nil)

	if entry.MessageCount != 0 {
		t.Errorf("expected MessageCount=0, got %d", entry.MessageCount)
	}
	if len(entry.FilesRead) != 0 {
		t.Errorf("expected empty FilesRead, got %v", entry.FilesRead)
	}
	if len(entry.FilesWritten) != 0 {
		t.Errorf("expected empty FilesWritten, got %v", entry.FilesWritten)
	}
}

// --- ShouldCompact tests ---

func TestShouldCompact_UnderBudget(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "hi"),
	}
	cfg := CompactionConfig{ReserveTokens: 16384}

	if ShouldCompact(msgs, 200000, cfg) {
		t.Error("ShouldCompact should be false when well under budget")
	}
}

func TestShouldCompact_OverBudget(t *testing.T) {
	// Create messages with ~60K chars → ~15K tokens
	largeText := strings.Repeat("word ", 12000)
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, largeText),
		ai.NewTextMessage(ai.RoleAssistant, largeText),
	}
	// Set context window to 20K tokens with 16K reserve → only 4K budget
	if !ShouldCompact(msgs, 20000, CompactionConfig{ReserveTokens: 16384}) {
		t.Error("ShouldCompact should be true when over budget")
	}
}

// --- FindCutPoint tests ---

func TestFindCutPoint_KeepsRecentTokens(t *testing.T) {
	msgs := make([]ai.Message, 20)
	for i := range msgs {
		role := ai.RoleUser
		if i%2 == 1 {
			role = ai.RoleAssistant
		}
		msgs[i] = ai.NewTextMessage(role, strings.Repeat("a", 100)) // ~25 tokens each
	}

	// Keep ~200 tokens worth → last ~8 messages
	cutIdx := FindCutPoint(msgs, 200)

	if cutIdx < 1 {
		t.Errorf("FindCutPoint should cut at least 1 message; got cutIdx=%d", cutIdx)
	}
	if cutIdx >= len(msgs) {
		t.Errorf("FindCutPoint should keep some messages; got cutIdx=%d out of %d", cutIdx, len(msgs))
	}
}

func TestFindCutPoint_NeverCutsMidToolResult(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, strings.Repeat("a", 400)),
		{Role: ai.RoleAssistant, Content: []ai.Content{
			{Type: ai.ContentToolUse, Name: "Read", Input: json.RawMessage(`{"file_path":"/tmp"}`)},
		}},
		{Role: ai.RoleUser, Content: []ai.Content{
			{Type: ai.ContentToolResult, ResultText: "file contents"},
		}},
		ai.NewTextMessage(ai.RoleAssistant, "here is the result"),
	}

	// Keep very few tokens to force a cut
	cutIdx := FindCutPoint(msgs, 50)

	// Must not cut between tool_use and tool_result (indices 1-2)
	// Valid cut points: 0 (before everything) or after the tool_result pair
	if cutIdx == 2 {
		t.Error("FindCutPoint should not cut between tool_use and tool_result")
	}
}

func TestFindCutPoint_FewMessages(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hi"),
	}

	cutIdx := FindCutPoint(msgs, 1000)
	if cutIdx != 0 {
		t.Errorf("FindCutPoint with 1 message should return 0; got %d", cutIdx)
	}
}

// --- CompactWithLLM tests ---

func TestCompactWithLLM_UsesInjectedSummarizer(t *testing.T) {
	msgs := make([]ai.Message, 20)
	for i := range msgs {
		role := ai.RoleUser
		if i%2 == 1 {
			role = ai.RoleAssistant
		}
		msgs[i] = ai.NewTextMessage(role, strings.Repeat("x", 100))
	}

	summarizer := func(_ context.Context, _ []ai.Message, _ string) (string, error) {
		return "test summary of conversation", nil
	}

	cfg := CompactionConfig{
		ReserveTokens:   16384,
		KeepRecentTokens: 200,
	}

	result, err := CompactWithLLM(context.Background(), msgs, cfg, summarizer)
	if err != nil {
		t.Fatalf("CompactWithLLM returned error: %v", err)
	}

	if result.Summary != "test summary of conversation" {
		t.Errorf("Summary = %q; want 'test summary of conversation'", result.Summary)
	}
	if result.FirstKeptIndex < 1 {
		t.Errorf("FirstKeptIndex should be > 0; got %d", result.FirstKeptIndex)
	}
	if len(result.Messages) < 3 {
		t.Errorf("Messages should have at least summary + ack + kept; got %d", len(result.Messages))
	}

	// First message should be the summary
	if !strings.Contains(result.Messages[0].Content[0].Text, "test summary") {
		t.Error("First message should contain the summary")
	}
}
