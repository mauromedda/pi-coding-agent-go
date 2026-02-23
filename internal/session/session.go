// ABOUTME: AgentSession orchestrator: model selection, retry, compaction trigger
// ABOUTME: Manages the lifecycle of an agent interaction session

package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Session orchestrates a single agent interaction session.
type Session struct {
	ID            string
	Model         *ai.Model
	Provider      ai.ApiProvider
	Messages      []ai.Message
	Writer        *Writer
	CWD           string
	ContextWindow int              // model's context window size in tokens
	Compaction    CompactionConfig // compaction settings
}

// NewSession creates a new session with the given model and provider.
func NewSession(id string, model *ai.Model, provider ai.ApiProvider, cwd string) (*Session, error) {
	writer, err := NewWriter(id)
	if err != nil {
		return nil, fmt.Errorf("creating session writer: %w", err)
	}

	s := &Session{
		ID:       id,
		Model:    model,
		Provider: provider,
		Writer:   writer,
		CWD:      cwd,
	}

	// Write session start record
	if err := writer.WriteRecord(RecordSessionStart, SessionStartData{
		ID:    id,
		Model: model.ID,
		CWD:   cwd,
	}); err != nil {
		return nil, fmt.Errorf("writing session start: %w", err)
	}

	return s, nil
}

// AddUserMessage appends a user message and persists it.
func (s *Session) AddUserMessage(content string) error {
	msg := ai.NewTextMessage(ai.RoleUser, content)
	s.Messages = append(s.Messages, msg)

	return s.Writer.WriteRecord(RecordUser, UserData{Content: content})
}

// AddAssistantMessage appends an assistant message and persists it.
func (s *Session) AddAssistantMessage(msg *ai.AssistantMessage) error {
	// Extract text content
	var text strings.Builder
	for _, c := range msg.Content {
		if c.Type == ai.ContentText {
			text.WriteString(c.Text)
		}
	}

	s.Messages = append(s.Messages, ai.Message{
		Role:    ai.RoleAssistant,
		Content: msg.Content,
	})

	return s.Writer.WriteRecord(RecordAssistant, AssistantData{
		Content:    text.String(),
		Model:      msg.Model,
		Usage:      UsageData{Input: msg.Usage.InputTokens, Output: msg.Usage.OutputTokens},
		StopReason: string(msg.StopReason),
	})
}

// BuildSessionContext reconstructs ai.Messages from persisted JSONL records.
// If a compaction record exists, it uses the latest one: a summary user message,
// an acknowledgment assistant message, then messages from kept records after
// the compaction. Without compaction, it rebuilds all user/assistant messages.
func BuildSessionContext(records []Record) ([]ai.Message, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Find the latest compaction record index.
	lastCompactionIdx := -1
	for i, rec := range records {
		if rec.Type == RecordCompaction {
			lastCompactionIdx = i
		}
	}

	if lastCompactionIdx >= 0 {
		return buildFromCompaction(records, lastCompactionIdx)
	}
	return buildFromAll(records)
}

// buildFromCompaction creates messages from a compaction point onward.
func buildFromCompaction(records []Record, compactionIdx int) ([]ai.Message, error) {
	var cd CompactionData
	if err := records[compactionIdx].Unmarshal(&cd); err != nil {
		return nil, fmt.Errorf("unmarshaling compaction data: %w", err)
	}

	// Build summary + file tracking text.
	summaryText := fmt.Sprintf("[Context Summary]\n%s", cd.Summary)
	if len(cd.FilesRead) > 0 {
		summaryText += "\n\n<read-files>\n"
		for _, f := range cd.FilesRead {
			summaryText += "- " + f + "\n"
		}
		summaryText += "</read-files>"
	}
	if len(cd.FilesWritten) > 0 {
		summaryText += "\n\n<modified-files>\n"
		for _, f := range cd.FilesWritten {
			summaryText += "- " + f + "\n"
		}
		summaryText += "</modified-files>"
	}
	summaryText += "\n[End Summary]"

	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, summaryText),
		ai.NewTextMessage(ai.RoleAssistant, "I understand the context. Let me continue from where we left off."),
	}

	// Append kept records after the compaction.
	kept := records[compactionIdx+1:]
	keptMsgs, err := buildFromAll(kept)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, keptMsgs...)
	return msgs, nil
}

// buildFromAll converts user and assistant records into ai.Messages.
func buildFromAll(records []Record) ([]ai.Message, error) {
	var msgs []ai.Message
	for _, rec := range records {
		switch rec.Type {
		case RecordUser:
			var ud UserData
			if err := rec.Unmarshal(&ud); err != nil {
				return nil, fmt.Errorf("unmarshaling user data: %w", err)
			}
			msgs = append(msgs, ai.NewTextMessage(ai.RoleUser, ud.Content))
		case RecordAssistant:
			var ad AssistantData
			if err := rec.Unmarshal(&ad); err != nil {
				return nil, fmt.Errorf("unmarshaling assistant data: %w", err)
			}
			msgs = append(msgs, ai.NewTextMessage(ai.RoleAssistant, ad.Content))
		}
	}
	return msgs, nil
}

// BuildContext creates the LLM context from current session state.
func (s *Session) BuildContext(systemPrompt string) *ai.Context {
	return &ai.Context{
		System:   systemPrompt,
		Messages: s.Messages,
	}
}

// NeedsCompaction returns true if the session's token usage exceeds the
// available budget (contextWindow - reserveTokens).
func (s *Session) NeedsCompaction() bool {
	if s.ContextWindow == 0 {
		return false // unknown context window; cannot determine
	}
	return ShouldCompact(s.Messages, s.ContextWindow, s.Compaction)
}

// Stream sends the current context to the LLM and returns an event stream.
func (s *Session) Stream(ctx context.Context, systemPrompt string, tools []ai.Tool, opts *ai.StreamOptions) *ai.EventStream {
	llmCtx := s.BuildContext(systemPrompt)
	llmCtx.Tools = tools
	return s.Provider.Stream(ctx, s.Model, llmCtx, opts)
}

// Close closes the session writer.
func (s *Session) Close() error {
	return s.Writer.Close()
}
