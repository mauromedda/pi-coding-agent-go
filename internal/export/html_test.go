// ABOUTME: Tests for HTML export functionality
// ABOUTME: Validates template rendering with various message types

package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestExportHTML_UserMessage(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "Hello, world!"),
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "<html") {
		t.Error("expected HTML document")
	}
	if !strings.Contains(out, "Hello, world!") {
		t.Error("expected user message text")
	}
	if !strings.Contains(out, "user") {
		t.Error("expected user role indicator")
	}
}

func TestExportHTML_AssistantMessage(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleAssistant, "I can help with that."),
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "I can help with that.") {
		t.Error("expected assistant message text")
	}
	if !strings.Contains(out, "assistant") {
		t.Error("expected assistant role indicator")
	}
}

func TestExportHTML_ToolUse(t *testing.T) {
	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					ID:    "tool_123",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/tmp/test.go"}`),
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Read") {
		t.Error("expected tool name")
	}
	if !strings.Contains(out, "file_path") {
		t.Error("expected tool input")
	}
}

func TestExportHTML_ToolResult(t *testing.T) {
	msgs := []ai.Message{
		{
			Role: ai.RoleUser,
			Content: []ai.Content{
				{
					Type:       ai.ContentToolResult,
					ID:         "tool_123",
					ResultText: "file contents here",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "<details") {
		t.Error("expected collapsible details element for tool result")
	}
	if !strings.Contains(out, "file contents here") {
		t.Error("expected tool result text")
	}
}

func TestExportHTML_ToolResultError(t *testing.T) {
	msgs := []ai.Message{
		{
			Role: ai.RoleUser,
			Content: []ai.Content{
				{
					Type:       ai.ContentToolResult,
					ID:         "tool_456",
					ResultText: "permission denied",
					IsError:    true,
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "permission denied") {
		t.Error("expected error text")
	}
	if !strings.Contains(out, "error") {
		t.Error("expected error indicator")
	}
}

func TestExportHTML_MultipleMessages(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "What is Go?"),
		ai.NewTextMessage(ai.RoleAssistant, "Go is a programming language."),
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "What is Go?") {
		t.Error("expected first message")
	}
	if !strings.Contains(out, "Go is a programming language.") {
		t.Error("expected second message")
	}
}

func TestExportHTML_EmptyMessages(t *testing.T) {
	var buf bytes.Buffer
	if err := ExportHTML(nil, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "<html") {
		t.Error("expected valid HTML even with no messages")
	}
}

func TestExportHTML_DarkThemeColors(t *testing.T) {
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "test"),
	}

	var buf bytes.Buffer
	if err := ExportHTML(msgs, &buf); err != nil {
		t.Fatalf("ExportHTML: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "#1e1e2e") {
		t.Error("expected dark theme background color")
	}
	if !strings.Contains(out, "#cdd6f4") {
		t.Error("expected dark theme text color")
	}
}
