// ABOUTME: Tests for the minion protocol (Distiller, Distributor, IngestDistill, CompressResult)
// ABOUTME: Uses mock providers to verify distillation, parallel extraction, ingest, and compression

package minion

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// mockProvider returns a canned assistant response for each Stream call.
type mockProvider struct {
	responses []*ai.AssistantMessage
	callCount atomic.Int32
	lastCtx   atomic.Pointer[ai.Context]
}

func (m *mockProvider) Api() ai.Api { return ai.ApiOpenAI }

func (m *mockProvider) Stream(_ context.Context, _ *ai.Model, ctx *ai.Context, _ *ai.StreamOptions) *ai.EventStream {
	m.lastCtx.Store(ctx)
	idx := int(m.callCount.Add(1)) - 1
	stream := ai.NewEventStream(16)

	go func() {
		if idx >= len(m.responses) {
			idx = len(m.responses) - 1
		}
		resp := m.responses[idx]
		for _, c := range resp.Content {
			if c.Type == ai.ContentText {
				stream.Send(ai.StreamEvent{Type: ai.EventContentDelta, Text: c.Text})
			}
		}
		stream.Finish(resp)
	}()

	return stream
}

var testModel = &ai.Model{ID: "llama3.2", Api: ai.ApiOpenAI, MaxOutputTokens: 4096}

func textMsg(text string) ai.Message {
	return ai.NewTextMessage(ai.RoleUser, text)
}

func assistantMsg(text string) ai.Message {
	return ai.Message{
		Role:    ai.RoleAssistant,
		Content: []ai.Content{{Type: ai.ContentText, Text: text}},
	}
}

func toolResultMsg(id, text string) ai.Message {
	return ai.Message{
		Role: ai.RoleUser,
		Content: []ai.Content{{
			Type:       ai.ContentToolResult,
			ID:         id,
			ResultText: text,
		}},
	}
}

func assistantToolCallMsg(toolName, toolID string) ai.Message {
	return ai.Message{
		Role: ai.RoleAssistant,
		Content: []ai.Content{{
			Type:  ai.ContentToolUse,
			ID:    toolID,
			Name:  toolName,
			Input: json.RawMessage(`{}`),
		}},
	}
}

// --- Distiller tests ---

func TestDistiller_Distill(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Distilled: user discussed file paths and type definitions."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider:   provider,
		Model:      testModel,
		KeepRecent: 2,
	})

	msgs := []ai.Message{
		textMsg("old message 1"),
		assistantMsg("old response 1"),
		textMsg("old message 2"),
		assistantMsg("old response 2"),
		textMsg("recent message"),
		assistantMsg("recent response"),
	}

	result, err := d.Distill(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// recentMsgs[0] is RoleUser, so summary merges into it: 2 messages total
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// First message should contain the merged summary + original user content
	first := result[0]
	if first.Role != ai.RoleUser {
		t.Errorf("first message role = %q, want %q", first.Role, ai.RoleUser)
	}
	if len(first.Content) < 2 {
		t.Fatalf("expected at least 2 content blocks (summary + original), got %d", len(first.Content))
	}
	if first.Content[0].Text == "" {
		t.Error("summary text should not be empty")
	}

	// Provider should have been called exactly once
	if provider.callCount.Load() != 1 {
		t.Errorf("provider called %d times, want 1", provider.callCount.Load())
	}

	// System prompt should be the distillation prompt
	if ctx := provider.lastCtx.Load(); ctx == nil || ctx.System != distillSystemPrompt {
		t.Error("expected distillation system prompt")
	}
}

func TestDistiller_FewMessages_NoDistillation(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "should not be called"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider:   provider,
		Model:      testModel,
		KeepRecent: 4,
	})

	msgs := []ai.Message{
		textMsg("msg1"),
		assistantMsg("resp1"),
	}

	result, err := d.Distill(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(msgs) {
		t.Errorf("expected %d messages unchanged, got %d", len(msgs), len(result))
	}
	if provider.callCount.Load() != 0 {
		t.Errorf("provider should not be called for short conversations, called %d times", provider.callCount.Load())
	}
}

func TestDistiller_ContextCancellation(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "summary"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider:   provider,
		Model:      testModel,
		KeepRecent: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msgs := []ai.Message{
		textMsg("old 1"),
		textMsg("old 2"),
		textMsg("recent"),
	}

	_, err := d.Distill(ctx, msgs)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestDistiller_DefaultConfig(t *testing.T) {
	t.Parallel()

	d := New(Config{
		Provider: &mockProvider{},
		Model:    testModel,
	})

	if d.config.KeepRecent != defaultKeepRecent {
		t.Errorf("KeepRecent = %d, want %d", d.config.KeepRecent, defaultKeepRecent)
	}
}

// --- Distributor tests ---

func TestDistributor_Distribute(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: `{"relevant_code": ["func Foo()"], "types": ["Bar"]}`}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := NewDistributor(DistributorConfig{
		Provider:   provider,
		Model:      testModel,
		MaxWorkers: 2,
		KeepRecent: 2,
	})

	msgs := []ai.Message{
		textMsg("read file X"),
		assistantMsg("here is file X content"),
		toolResultMsg("tc_1", "file content here"),
		textMsg("now edit Y"),
		assistantMsg("editing Y"),
		toolResultMsg("tc_2", "done editing"),
		textMsg("recent 1"),
		assistantMsg("recent 2"),
	}

	result, err := d.Distribute(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// recentMsgs[0] is RoleUser, so summary merges into it: 2 messages total
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// First message should contain the merged aggregated context + original user content
	first := result[0]
	if first.Role != ai.RoleUser {
		t.Errorf("first message role = %q, want %q", first.Role, ai.RoleUser)
	}
	if len(first.Content) < 2 {
		t.Fatalf("expected at least 2 content blocks (aggregated + original), got %d", len(first.Content))
	}
	if first.Content[0].Text == "" {
		t.Error("aggregated text should not be empty")
	}

	// Provider should have been called for each chunk
	if provider.callCount.Load() == 0 {
		t.Error("provider should have been called at least once")
	}
}

func TestDistributor_FewMessages_NoDistribution(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "nope"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := NewDistributor(DistributorConfig{
		Provider:   provider,
		Model:      testModel,
		KeepRecent: 4,
	})

	msgs := []ai.Message{
		textMsg("short"),
		assistantMsg("convo"),
	}

	result, err := d.Distribute(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(msgs) {
		t.Errorf("expected %d messages unchanged, got %d", len(msgs), len(result))
	}
	if provider.callCount.Load() != 0 {
		t.Error("provider should not be called for short conversations")
	}
}

func TestDistributor_ContextCancellation(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "data"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := NewDistributor(DistributorConfig{
		Provider:   provider,
		Model:      testModel,
		MaxWorkers: 2,
		KeepRecent: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msgs := []ai.Message{
		textMsg("old 1"),
		toolResultMsg("tc_1", "content"),
		textMsg("old 2"),
		toolResultMsg("tc_2", "content 2"),
		textMsg("recent"),
	}

	_, err := d.Distribute(ctx, msgs)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestSplitIntoChunks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		msgs       []ai.Message
		wantChunks int
	}{
		{
			name:       "empty",
			msgs:       nil,
			wantChunks: 0,
		},
		{
			name: "single tool result boundary",
			msgs: []ai.Message{
				textMsg("ask"),
				assistantToolCallMsg("read", "tc_1"),
				toolResultMsg("tc_1", "content"),
			},
			wantChunks: 1,
		},
		{
			name: "two tool result boundaries",
			msgs: []ai.Message{
				textMsg("ask"),
				toolResultMsg("tc_1", "content"),
				textMsg("ask again"),
				toolResultMsg("tc_2", "done"),
			},
			wantChunks: 2,
		},
		{
			name: "trailing messages without tool result",
			msgs: []ai.Message{
				textMsg("ask"),
				toolResultMsg("tc_1", "content"),
				textMsg("follow up"),
				assistantMsg("response"),
			},
			wantChunks: 2,
		},
		{
			name: "no tool results",
			msgs: []ai.Message{
				textMsg("ask"),
				assistantMsg("response"),
			},
			wantChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chunks := splitIntoChunks(tt.msgs)
			if len(chunks) != tt.wantChunks {
				t.Errorf("got %d chunks, want %d", len(chunks), tt.wantChunks)
			}
		})
	}
}

func TestAggregateExtracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		extracts []string
		wantLen  bool
	}{
		{"multiple extracts", []string{"chunk 1 data", "chunk 2 data", "chunk 3 data"}, true},
		{"single extract", []string{"only chunk"}, true},
		{"empty extracts skipped", []string{"", "data", ""}, true},
		{"all empty", []string{"", "", ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := aggregateExtracts(tt.extracts)
			if tt.wantLen && result == "" {
				t.Error("expected non-empty result")
			}
			if !tt.wantLen && result != "" {
				t.Errorf("expected empty result, got %q", result)
			}
		})
	}
}

func TestDistributor_DefaultConfig(t *testing.T) {
	t.Parallel()

	d := NewDistributor(DistributorConfig{
		Provider: &mockProvider{},
		Model:    testModel,
	})

	if d.config.MaxWorkers != defaultMaxWorkers {
		t.Errorf("MaxWorkers = %d, want %d", d.config.MaxWorkers, defaultMaxWorkers)
	}
	if d.config.KeepRecent != defaultKeepRecent {
		t.Errorf("KeepRecent = %d, want %d", d.config.KeepRecent, defaultKeepRecent)
	}
}

// --- Role alternation tests ---

// assertRoleAlternation fails the test if any two consecutive messages share the same role.
func assertRoleAlternation(t *testing.T, msgs []ai.Message) {
	t.Helper()
	for i := 1; i < len(msgs); i++ {
		if msgs[i].Role == msgs[i-1].Role {
			t.Errorf("role alternation violated at index %d: %s followed by %s",
				i, msgs[i-1].Role, msgs[i].Role)
		}
	}
}

func TestDistiller_RoleAlternation(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Summary of old conversation."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider:   provider,
		Model:      testModel,
		KeepRecent: 2,
	})

	// Recent messages start with user (common case that triggered the bug)
	msgs := []ai.Message{
		textMsg("old message 1"),
		assistantMsg("old response 1"),
		textMsg("old message 2"),
		assistantMsg("old response 2"),
		textMsg("recent user msg"),      // RoleUser
		assistantMsg("recent assistant"), // RoleAssistant
	}

	result, err := d.Distill(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertRoleAlternation(t, result)

	// Summary should be merged into the first user message
	if len(result) != 2 {
		t.Fatalf("expected 2 messages (merged summary+user, assistant), got %d", len(result))
	}
	if result[0].Role != ai.RoleUser {
		t.Errorf("first message role = %q, want %q", result[0].Role, ai.RoleUser)
	}
	// First content block should be the summary
	if len(result[0].Content) < 2 {
		t.Fatal("expected summary content merged with original user content")
	}
	if result[0].Content[0].Text == "" || result[0].Content[0].Type != ai.ContentText {
		t.Error("first content block should be the summary text")
	}
}

func TestDistiller_RoleAlternation_AssistantFirst(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Summary."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider:   provider,
		Model:      testModel,
		KeepRecent: 2,
	})

	// Recent messages start with assistant (summary can be a separate user message)
	msgs := []ai.Message{
		textMsg("old message 1"),
		assistantMsg("old response 1"),
		textMsg("old message 2"),
		assistantMsg("recent assistant"), // RoleAssistant (first of recent)
		textMsg("recent user msg"),       // RoleUser
	}

	result, err := d.Distill(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertRoleAlternation(t, result)

	// Summary prepended as separate message: [user:summary, assistant, user]
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != ai.RoleUser {
		t.Errorf("first message role = %q, want %q", result[0].Role, ai.RoleUser)
	}
}

func TestDistributor_RoleAlternation(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: `{"files": ["main.go"], "types": ["Config"]}`}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := NewDistributor(DistributorConfig{
		Provider:   provider,
		Model:      testModel,
		MaxWorkers: 2,
		KeepRecent: 2,
	})

	// Recent messages start with user (triggers the bug)
	msgs := []ai.Message{
		textMsg("read file X"),
		assistantMsg("here is file X content"),
		toolResultMsg("tc_1", "file content here"),
		textMsg("now edit Y"),
		assistantMsg("editing Y"),
		toolResultMsg("tc_2", "done editing"),
		textMsg("recent user msg"),      // RoleUser
		assistantMsg("recent assistant"), // RoleAssistant
	}

	result, err := d.Distribute(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertRoleAlternation(t, result)

	// Summary merged into user message: [user:aggregated+original, assistant]
	if len(result) != 2 {
		t.Fatalf("expected 2 messages (merged, assistant), got %d", len(result))
	}
}

func TestPrependSummary(t *testing.T) {
	t.Parallel()

	summary := ai.Content{Type: ai.ContentText, Text: "[Summary] test summary"}

	tests := []struct {
		name       string
		recent     []ai.Message
		wantLen    int
		wantMerged bool // true if summary should be merged into first message
	}{
		{
			name:       "empty recent",
			recent:     nil,
			wantLen:    1,
			wantMerged: false,
		},
		{
			name:       "first message is user (merge)",
			recent:     []ai.Message{textMsg("hello"), assistantMsg("hi")},
			wantLen:    2,
			wantMerged: true,
		},
		{
			name:       "first message is assistant (separate)",
			recent:     []ai.Message{assistantMsg("hi"), textMsg("hello")},
			wantLen:    3,
			wantMerged: false,
		},
		{
			name:       "single user message (merge)",
			recent:     []ai.Message{textMsg("only msg")},
			wantLen:    1,
			wantMerged: true,
		},
		{
			name:       "single assistant message (separate)",
			recent:     []ai.Message{assistantMsg("only msg")},
			wantLen:    2,
			wantMerged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := prependSummary(summary, tt.recent)

			if len(result) != tt.wantLen {
				t.Errorf("got %d messages, want %d", len(result), tt.wantLen)
			}

			// First message must always be user
			if len(result) > 0 && result[0].Role != ai.RoleUser {
				t.Errorf("first message role = %q, want %q", result[0].Role, ai.RoleUser)
			}

			// Verify role alternation
			assertRoleAlternation(t, result)

			if tt.wantMerged {
				// Merged: first content block is the summary, rest is original
				if len(result[0].Content) < 2 {
					t.Errorf("expected merged content blocks, got %d", len(result[0].Content))
				}
			}
		})
	}
}

// --- IngestDistill tests ---

func TestDistiller_IngestDistill(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Full context summary."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider: provider,
		Model:    testModel,
	})

	msgs := []ai.Message{
		textMsg("first message"),
		assistantMsg("first response"),
		textMsg("second message"),
		assistantMsg("second response"),
	}

	summary, err := d.IngestDistill(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call, got %d", provider.callCount.Load())
	}
}

func TestDistiller_IngestDistill_Empty(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "should not be called"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider: provider,
		Model:    testModel,
	})

	summary, err := d.IngestDistill(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary for nil messages, got %q", summary)
	}
	if provider.callCount.Load() != 0 {
		t.Errorf("expected 0 provider calls, got %d", provider.callCount.Load())
	}
}

// --- CompressResult tests ---

func TestDistiller_CompressResult_Short(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "should not be called"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider: provider,
		Model:    testModel,
	})

	text := "short result"
	result, err := d.CompressResult(context.Background(), text, 4000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != text {
		t.Errorf("expected passthrough, got %q", result)
	}
	if provider.callCount.Load() != 0 {
		t.Errorf("expected 0 provider calls for short text, got %d", provider.callCount.Load())
	}
}

func TestDistiller_CompressResult_Long(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Compressed: key findings preserved."}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	d := New(Config{
		Provider: provider,
		Model:    testModel,
	})

	longText := strings.Repeat("verbose output from sub-agent. ", 200)
	result, err := d.CompressResult(context.Background(), longText, 4000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty compressed result")
	}
	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call, got %d", provider.callCount.Load())
	}

	// Verify the system prompt used was the compression prompt
	ctx := provider.lastCtx.Load()
	if ctx == nil {
		t.Fatal("provider context was not captured")
	}
	if ctx.System != compressResultSystemPrompt {
		t.Error("expected compression system prompt")
	}
}
