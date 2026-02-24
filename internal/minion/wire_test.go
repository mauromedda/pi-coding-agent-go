// ABOUTME: Tests for BuildTransform factory wiring minion protocol from CLI config
// ABOUTME: Verifies singular/plural mode selection and provider passthrough

package minion

import (
	"context"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBuildTransform_Singular(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "distilled"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	transform := BuildTransform(WireConfig{
		Mode:     ModeSingular,
		Model:    testModel,
		Provider: provider,
	})

	msgs := []ai.Message{
		textMsg("old 1"),
		assistantMsg("old 2"),
		textMsg("old 3"),
		assistantMsg("old 4"),
		textMsg("old 5"),
		assistantMsg("recent"),
	}

	result, err := transform(context.Background(), msgs)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	// Should produce summary + recent messages
	if len(result) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(result))
	}

	first := result[0]
	if first.Role != ai.RoleUser {
		t.Fatalf("first is %s, want %s", first.Role, ai.RoleUser)
	}
	if !strings.Contains(first.Content[0].Text, "Prior conversation summary") {
		t.Errorf("expected summary marker, got %q", first.Content[0].Text)
	}
}

func TestBuildTransform_Plural(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "extracted"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	transform := BuildTransform(WireConfig{
		Mode:     ModePlural,
		Model:    testModel,
		Provider: provider,
	})

	msgs := []ai.Message{
		textMsg("ask"),
		assistantMsg("response"),
		toolResultMsg("tc_1", "content"),
		textMsg("ask 2"),
		assistantMsg("response 2"),
		toolResultMsg("tc_2", "done"),
		textMsg("recent 1"),
		assistantMsg("recent 2"),
		textMsg("recent 3"),
		assistantMsg("recent 4"),
	}

	result, err := transform(context.Background(), msgs)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	first := result[0]
	if first.Role != ai.RoleUser {
		t.Fatalf("first is %s, want %s", first.Role, ai.RoleUser)
	}
	if !strings.Contains(first.Content[0].Text, "Aggregated Context") {
		t.Errorf("expected Aggregated Context marker, got %q", first.Content[0].Text)
	}
}

func TestBuildTransform_DefaultIsSingular(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "summary"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	// Empty mode string should default to singular
	transform := BuildTransform(WireConfig{
		Mode:     "",
		Model:    testModel,
		Provider: provider,
	})

	msgs := []ai.Message{
		textMsg("old 1"),
		assistantMsg("old 2"),
		textMsg("old 3"),
		assistantMsg("old 4"),
		textMsg("old 5"),
		assistantMsg("recent"),
	}

	result, err := transform(context.Background(), msgs)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	first := result[0]
	if first.Role != ai.RoleUser {
		t.Fatalf("first is %s, want %s", first.Role, ai.RoleUser)
	}
	if !strings.Contains(first.Content[0].Text, "Prior conversation summary") {
		t.Errorf("expected summary marker (singular default), got %q", first.Content[0].Text)
	}
}
