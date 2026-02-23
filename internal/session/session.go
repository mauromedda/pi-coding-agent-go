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
