// ABOUTME: Tests for file operation reversal logic (revert/undo commands)
// ABOUTME: Covers FindFileOps extraction from messages and operation classification

package revert

import (
	"encoding/json"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestFindFileOps_Empty(t *testing.T) {
	t.Parallel()

	ops := FindFileOps(nil, 10)
	if len(ops) != 0 {
		t.Errorf("expected 0 ops, got %d", len(ops))
	}
}

func TestFindFileOps_ExtractsWriteTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "write",
					Input: json.RawMessage(`{"path": "/tmp/test.go", "content": "package main"}`),
				},
			},
		},
	}

	ops := FindFileOps(msgs, 10)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Path != "/tmp/test.go" {
		t.Errorf("expected path '/tmp/test.go', got %q", ops[0].Path)
	}
	if ops[0].Tool != "write" {
		t.Errorf("expected tool 'write', got %q", ops[0].Tool)
	}
}

func TestFindFileOps_ExtractsEditTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "edit",
					Input: json.RawMessage(`{"path": "/tmp/edit.go", "old_text": "a", "new_text": "b"}`),
				},
			},
		},
	}

	ops := FindFileOps(msgs, 10)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Path != "/tmp/edit.go" {
		t.Errorf("expected path '/tmp/edit.go', got %q", ops[0].Path)
	}
}

func TestFindFileOps_RespectsMaxSteps(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{Type: ai.ContentToolUse, Name: "write", Input: json.RawMessage(`{"path": "/a.go"}`)},
				{Type: ai.ContentToolUse, Name: "write", Input: json.RawMessage(`{"path": "/b.go"}`)},
				{Type: ai.ContentToolUse, Name: "write", Input: json.RawMessage(`{"path": "/c.go"}`)},
			},
		},
	}

	ops := FindFileOps(msgs, 2)
	if len(ops) != 2 {
		t.Errorf("expected 2 ops (maxSteps=2), got %d", len(ops))
	}
}

func TestFindFileOps_SkipsNonFileTools(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{Type: ai.ContentToolUse, Name: "read", Input: json.RawMessage(`{"path": "/a.go"}`)},
				{Type: ai.ContentText, Text: "some text"},
			},
		},
	}

	ops := FindFileOps(msgs, 10)
	if len(ops) != 0 {
		t.Errorf("expected 0 ops for non-file tools, got %d", len(ops))
	}
}

func TestFindFileOps_WalksBackward(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{Type: ai.ContentToolUse, Name: "write", Input: json.RawMessage(`{"path": "/first.go"}`)},
			},
		},
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{Type: ai.ContentToolUse, Name: "write", Input: json.RawMessage(`{"path": "/second.go"}`)},
			},
		},
	}

	ops := FindFileOps(msgs, 1)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	// Should find the most recent (last) message first
	if ops[0].Path != "/second.go" {
		t.Errorf("expected '/second.go' (most recent), got %q", ops[0].Path)
	}
}
