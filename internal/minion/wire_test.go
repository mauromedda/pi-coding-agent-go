// ABOUTME: Tests for BuildIngester and BuildResultCompressor factory functions
// ABOUTME: Verifies ingest and compression wiring with mock provider

package minion

import (
	"context"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBuildIngester(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Ingested summary of conversation."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	ingester := BuildIngester(WireConfig{
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

	summary, err := ingester(context.Background(), msgs)
	if err != nil {
		t.Fatalf("ingester: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call, got %d", provider.callCount.Load())
	}
}

func TestBuildIngester_EmptyMessages(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "should not be called"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	ingester := BuildIngester(WireConfig{
		Model:    testModel,
		Provider: provider,
	})

	summary, err := ingester(context.Background(), nil)
	if err != nil {
		t.Fatalf("ingester: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary for nil messages, got %q", summary)
	}
	if provider.callCount.Load() != 0 {
		t.Errorf("expected 0 provider calls for empty messages, got %d", provider.callCount.Load())
	}
}

func TestBuildResultCompressor(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Compressed output."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	compressor := BuildResultCompressor(WireConfig{
		Model:    testModel,
		Provider: provider,
	})

	longText := strings.Repeat("x", 5000)
	compressed, err := compressor(context.Background(), longText, 4000)
	if err != nil {
		t.Fatalf("compressor: %v", err)
	}
	if compressed == "" {
		t.Error("expected non-empty compressed result")
	}
	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call, got %d", provider.callCount.Load())
	}
}

func TestBuildResultCompressor_ShortText(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "should not be called"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	compressor := BuildResultCompressor(WireConfig{
		Model:    testModel,
		Provider: provider,
	})

	shortText := "short result"
	result, err := compressor(context.Background(), shortText, 4000)
	if err != nil {
		t.Fatalf("compressor: %v", err)
	}
	if result != shortText {
		t.Errorf("expected passthrough for short text, got %q", result)
	}
	if provider.callCount.Load() != 0 {
		t.Errorf("expected 0 provider calls for short text, got %d", provider.callCount.Load())
	}
}
