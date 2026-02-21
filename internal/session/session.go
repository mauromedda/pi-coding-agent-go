// ABOUTME: AgentSession orchestrator: model selection, retry, compaction trigger
// ABOUTME: Manages the lifecycle of an agent interaction session

package session

import (
	"context"
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

const compactionThreshold = 100 // Messages before auto-compaction

// Session orchestrates a single agent interaction session.
type Session struct {
	ID       string
	Model    *ai.Model
	Provider ai.ApiProvider
	Messages []ai.Message
	Writer   *Writer
	CWD      string
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
	var text string
	for _, c := range msg.Content {
		if c.Type == ai.ContentText {
			text += c.Text
		}
	}

	s.Messages = append(s.Messages, ai.Message{
		Role:    ai.RoleAssistant,
		Content: msg.Content,
	})

	return s.Writer.WriteRecord(RecordAssistant, AssistantData{
		Content:    text,
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

// NeedsCompaction returns true if the session has exceeded the compaction threshold.
func (s *Session) NeedsCompaction() bool {
	return len(s.Messages) >= compactionThreshold
}

// Stream sends the current context to the LLM and returns an event stream.
func (s *Session) Stream(ctx context.Context, systemPrompt string, tools []ai.Tool, opts *ai.StreamOptions) *ai.EventStream {
	llmCtx := s.BuildContext(systemPrompt)
	llmCtx.Tools = tools
	_ = ctx // Context propagation handled by provider
	return s.Provider.Stream(s.Model, llmCtx, opts)
}

// Close closes the session writer.
func (s *Session) Close() error {
	return s.Writer.Close()
}
