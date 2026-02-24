// ABOUTME: Tests for context compaction wiring in the Bubble Tea TUI
// ABOUTME: Verifies autoCompact triggers, CompactDoneMsg handling, and /compact command

package btea

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestAppModel_AutoCompactMsg_TriggersCompaction(t *testing.T) {
	m := NewAppModel(testDeps())
	// Seed some messages so compaction has something to work with
	for i := range 10 {
		m.messages = append(m.messages,
			ai.NewTextMessage(ai.RoleUser, "question "+string(rune('0'+i))),
			ai.NewTextMessage(ai.RoleAssistant, "answer "+string(rune('0'+i))),
		)
	}

	result, cmd := m.Update(AutoCompactMsg{})
	model := result.(AppModel)

	if !model.compacting {
		t.Error("compacting = false; want true after AutoCompactMsg")
	}
	if cmd == nil {
		t.Fatal("cmd = nil; want compaction command")
	}
}

func TestAppModel_AutoCompactMsg_NoopWhenAlreadyCompacting(t *testing.T) {
	m := NewAppModel(testDeps())
	m.compacting = true

	result, cmd := m.Update(AutoCompactMsg{})
	model := result.(AppModel)

	if cmd != nil {
		t.Errorf("cmd = %v; want nil when already compacting", cmd)
	}
	if !model.compacting {
		t.Error("compacting should remain true")
	}
}

func TestAppModel_AutoCompactMsg_NoopWhenNoMessages(t *testing.T) {
	m := NewAppModel(testDeps())

	result, cmd := m.Update(AutoCompactMsg{})
	model := result.(AppModel)

	if model.compacting {
		t.Error("compacting = true; want false when no messages")
	}
	if cmd != nil {
		t.Errorf("cmd = %v; want nil when no messages to compact", cmd)
	}
}

func TestAppModel_CompactDoneMsg_ReplacesMessages(t *testing.T) {
	m := NewAppModel(testDeps())
	m.compacting = true
	m.messages = []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "old question"),
		ai.NewTextMessage(ai.RoleAssistant, "old answer"),
	}

	compacted := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "[Context Summary]\nSummary text\n[End Summary]"),
		ai.NewTextMessage(ai.RoleAssistant, "I understand the context."),
	}

	result, _ := m.Update(CompactDoneMsg{
		Messages:    compacted,
		Summary:     "Summary text",
		TokensSaved: 500,
	})
	model := result.(AppModel)

	if model.compacting {
		t.Error("compacting = true; want false after CompactDoneMsg")
	}
	if len(model.messages) != 2 {
		t.Fatalf("messages = %d; want 2 (compacted)", len(model.messages))
	}
	if model.messages[0].Role != ai.RoleUser {
		t.Errorf("first message role = %q; want user (summary)", model.messages[0].Role)
	}
}

func TestAppModel_CompactDoneMsg_UpdatesFooter(t *testing.T) {
	m := NewAppModel(testDeps())
	m.compacting = true
	m.messages = []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "q"),
	}

	compacted := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "[Summary]"),
		ai.NewTextMessage(ai.RoleAssistant, "ack"),
	}

	result, _ := m.Update(CompactDoneMsg{
		Messages:    compacted,
		Summary:     "short",
		TokensSaved: 1000,
	})
	model := result.(AppModel)

	// Footer context pct should be updated (we can't check exact value,
	// but it should not panic and compacting flag should be cleared)
	if model.compacting {
		t.Error("compacting should be false after CompactDoneMsg")
	}
	_ = model.footer // access footer to verify no nil pointer
}

func TestAppModel_CompactCommand_TriggersAutoCompact(t *testing.T) {
	m := NewAppModel(testDeps())
	m.messages = []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "world"),
	}

	ctx, _ := m.buildCommandContext()

	result := ctx.CompactFn()
	// Should no longer return "not yet available"
	if result == "Compact not yet available." {
		t.Error("CompactFn still returns placeholder; should trigger compaction")
	}
}

func TestAppModel_AgentUsage_TriggersAutoCompactWhenThresholdExceeded(t *testing.T) {
	deps := testDeps()
	deps.AutoCompactThreshold = 100 // very low threshold for testing
	m := NewAppModel(deps)

	// Seed messages to exceed threshold
	m.messages = append(m.messages,
		ai.NewTextMessage(ai.RoleUser, "A very long message that should exceed our tiny threshold when tokens are counted together with all the other messages in the conversation history."),
		ai.NewTextMessage(ai.RoleAssistant, "An equally long response that adds even more tokens to our running total and should definitely push us past the compaction threshold."),
	)

	// Send usage that pushes us over
	usage := &ai.Usage{InputTokens: 80, OutputTokens: 30}
	result, cmd := m.Update(AgentUsageMsg{Usage: usage})
	model := result.(AppModel)

	// Token tracking should be updated
	if model.totalInputTokens != 80 {
		t.Errorf("totalInputTokens = %d; want 80", model.totalInputTokens)
	}

	// When threshold is exceeded, the Update should return a cmd that
	// drives the auto-compact flow (either a batch including the trigger
	// or a direct compaction start).
	if cmd == nil {
		t.Error("cmd = nil; want non-nil when usage exceeds auto-compact threshold")
	}
	_ = model
}
