// ABOUTME: Tests for pre-flight compaction and adaptive MaxOutputTokens in the agent loop
// ABOUTME: Verifies compaction triggers before Stream and output tokens adapt to budget

package agent

import (
	"context"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/perf"
	"github.com/mauromedda/pi-coding-agent-go/internal/session"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestAdaptiveMaxOutputTokens_SmallInput(t *testing.T) {
	t.Parallel()

	profile := perf.ModelProfile{
		ContextWindow:   200000,
		MaxOutputTokens: 4096,
		Latency:         perf.LatencyFast,
	}

	// Small input: MaxOutputTokens should stay at model default
	msgs := []ai.Message{ai.NewTextMessage(ai.RoleUser, "hello")}
	inputTokens := session.EstimateMessagesTokens(msgs)
	params := perf.Decide(profile, inputTokens, 200000)

	if params.MaxOutputTokens != 4096 {
		t.Errorf("MaxOutputTokens = %d, want 4096 (model default)", params.MaxOutputTokens)
	}
}

func TestAdaptiveMaxOutputTokens_LargeInput(t *testing.T) {
	t.Parallel()

	profile := perf.ModelProfile{
		ContextWindow:   32000,
		MaxOutputTokens: 4096,
		Latency:         perf.LatencyLocal,
	}

	// Large input: MaxOutputTokens should shrink
	bigText := make([]byte, 120000) // ~30000 tokens at 4 bytes/token
	for i := range bigText {
		bigText[i] = 'a'
	}
	msgs := []ai.Message{ai.NewTextMessage(ai.RoleUser, string(bigText))}
	inputTokens := session.EstimateMessagesTokens(msgs)
	params := perf.Decide(profile, inputTokens, 32000)

	if params.MaxOutputTokens >= 4096 {
		t.Errorf("MaxOutputTokens = %d, should be < 4096 with large input", params.MaxOutputTokens)
	}
	if params.MaxOutputTokens < 1024 {
		t.Errorf("MaxOutputTokens = %d, should not drop below minimum floor", params.MaxOutputTokens)
	}
}

func TestPreflightCompaction_TriggersAboveThreshold(t *testing.T) {
	t.Parallel()

	contextWindow := 10000
	cfg := session.CompactionConfig{
		ReserveTokens:    2000,
		KeepRecentTokens: 3000,
	}

	// Create messages that exceed the budget
	bigText := make([]byte, 36000) // ~9000 tokens > budget of 8000
	for i := range bigText {
		bigText[i] = 'x'
	}
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "initial context"),
		ai.NewTextMessage(ai.RoleAssistant, "initial response"),
		ai.NewTextMessage(ai.RoleUser, string(bigText)),
	}

	shouldCompact := session.ShouldCompact(msgs, contextWindow, cfg)
	if !shouldCompact {
		t.Error("expected ShouldCompact = true when messages exceed budget")
	}
}

func TestPreflightCompaction_NotTriggeredBelowThreshold(t *testing.T) {
	t.Parallel()

	contextWindow := 100000
	cfg := session.CompactionConfig{
		ReserveTokens:    2000,
		KeepRecentTokens: 3000,
	}

	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "hi there"),
	}

	shouldCompact := session.ShouldCompact(msgs, contextWindow, cfg)
	if shouldCompact {
		t.Error("expected ShouldCompact = false for small messages")
	}
}

func TestAgentWithAdaptive_OptsApplied(t *testing.T) {
	t.Parallel()

	var capturedOpts *ai.StreamOptions
	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "response"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	// Override Stream to capture opts
	capturingProvider := &optsCapturingProvider{
		inner:        provider,
		capturedOpts: &capturedOpts,
	}

	model := &ai.Model{
		ID: "test", Name: "Test", Api: ai.ApiAnthropic,
		MaxOutputTokens: 4096, ContextWindow: 200000,
	}

	ag := New(capturingProvider, model, nil)
	ag.adaptive = &AdaptiveConfig{
		Profile: perf.ModelProfile{
			ContextWindow:         200000,
			MaxOutputTokens:       4096,
			SupportsPromptCaching: true,
			Latency:               perf.LatencyFast,
		},
	}

	llmCtx := newTestContext()
	opts := &ai.StreamOptions{MaxTokens: 8192} // intentionally high
	_ = collectEvents(ag.Prompt(context.Background(), llmCtx, opts))

	if capturedOpts == nil {
		t.Fatal("expected Stream to be called with options")
	}
	// Should be clamped to model's MaxOutputTokens since input is small
	if capturedOpts.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096 (clamped to model max)", capturedOpts.MaxTokens)
	}
}

// optsCapturingProvider wraps a provider to capture StreamOptions.
type optsCapturingProvider struct {
	inner        ai.ApiProvider
	capturedOpts **ai.StreamOptions
}

func (p *optsCapturingProvider) Api() ai.Api { return p.inner.Api() }

func (p *optsCapturingProvider) Stream(ctx context.Context, model *ai.Model, llmCtx *ai.Context, opts *ai.StreamOptions) *ai.EventStream {
	cp := *opts
	*p.capturedOpts = &cp
	return p.inner.Stream(ctx, model, llmCtx, opts)
}
