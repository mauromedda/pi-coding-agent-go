// ABOUTME: Distiller implements the singular minion protocol (sequential distillation)
// ABOUTME: Local model reads full context, produces a compressed summary for the cloud model

package minion

import (
	"context"
	"fmt"
	"strings"

	pilog "github.com/mauromedda/pi-coding-agent-go/internal/log"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

const (
	defaultKeepRecent     = 4
	defaultLocalMaxTokens = 4096
)

// Config holds the configuration for a Distiller.
type Config struct {
	Provider   ai.ApiProvider
	Model      *ai.Model
	KeepRecent int // number of recent messages to pass through unmodified (default 4)
}

// Distiller implements the singular minion protocol: sends the full message
// list to a local model, which extracts relevant context. Returns a reduced
// message list (summary + recent messages).
type Distiller struct {
	config Config
}

// New creates a Distiller with the given configuration.
func New(cfg Config) *Distiller {
	if cfg.KeepRecent <= 0 {
		cfg.KeepRecent = defaultKeepRecent
	}
	return &Distiller{config: cfg}
}

// Distill sends the full message list to the local model for distillation.
// Returns: [SummaryMessage, ...recentMessages]. If there are fewer messages
// than KeepRecent, returns the original messages unchanged.
func (d *Distiller) Distill(ctx context.Context, msgs []ai.Message) ([]ai.Message, error) {
	pilog.Debug("minion/singular: distilling %d messages (keep_recent=%d)", len(msgs), d.config.KeepRecent)

	if len(msgs) <= d.config.KeepRecent {
		pilog.Debug("minion/singular: skipping, only %d messages (threshold %d)", len(msgs), d.config.KeepRecent)
		return msgs, nil
	}

	olderMsgs := msgs[:len(msgs)-d.config.KeepRecent]
	recentMsgs := msgs[len(msgs)-d.config.KeepRecent:]

	pilog.Debug("minion/singular: summarizing %d older messages, keeping %d recent", len(olderMsgs), len(recentMsgs))

	summary, err := d.distillMessages(ctx, olderMsgs)
	if err != nil {
		return nil, fmt.Errorf("distill: %w", err)
	}

	pilog.Debug("minion/singular: summary length=%d chars", len(summary))

	summaryContent := ai.Content{Type: ai.ContentText, Text: "[Prior conversation summary]\n" + summary}
	return prependSummary(summaryContent, recentMsgs), nil
}

// prependSummary inserts a summary content block before recentMsgs while
// preserving strict role alternation. If the first recent message is already
// RoleUser, the summary is merged into that message's content. Otherwise a
// new user message is prepended.
func prependSummary(summary ai.Content, recentMsgs []ai.Message) []ai.Message {
	if len(recentMsgs) > 0 && recentMsgs[0].Role == ai.RoleUser {
		// Merge summary into the existing user message to avoid consecutive user messages.
		combined := make([]ai.Content, 0, 1+len(recentMsgs[0].Content))
		combined = append(combined, summary)
		combined = append(combined, recentMsgs[0].Content...)
		result := make([]ai.Message, 0, len(recentMsgs))
		result = append(result, ai.Message{Role: ai.RoleUser, Content: combined})
		result = append(result, recentMsgs[1:]...)
		return result
	}

	// Safe to prepend as a separate message (next message is assistant or empty).
	summaryMsg := ai.Message{
		Role:    ai.RoleUser,
		Content: []ai.Content{summary},
	}
	return append([]ai.Message{summaryMsg}, recentMsgs...)
}

// IngestDistill performs one-shot distillation of the full message list.
// Unlike Distill (which splits recent/old), this compresses everything
// into a single summary string. Used before the first agent turn.
func (d *Distiller) IngestDistill(ctx context.Context, msgs []ai.Message) (string, error) {
	if len(msgs) == 0 {
		return "", nil
	}
	pilog.Debug("minion/ingest: distilling %d messages", len(msgs))
	return d.distillMessages(ctx, msgs)
}

// CompressResult distills a sub-agent's text result into a shorter summary.
// Returns the original text unchanged if it's already within maxLen characters.
func (d *Distiller) CompressResult(ctx context.Context, text string, maxLen int) (string, error) {
	if len(text) <= maxLen {
		return text, nil
	}
	pilog.Debug("minion/compress: compressing %d chars (max=%d)", len(text), maxLen)
	compressed, err := d.callLocalModel(ctx, compressResultSystemPrompt, compressResultUserPromptPrefix+text)
	if err != nil {
		return "", fmt.Errorf("compress result: %w", err)
	}
	pilog.Debug("minion/compress: %d -> %d chars", len(text), len(compressed))
	return compressed, nil
}

// distillMessages calls the local model to produce a summary of the given messages.
func (d *Distiller) distillMessages(ctx context.Context, msgs []ai.Message) (string, error) {
	conversationText := formatMessages(msgs)
	pilog.Debug("minion/singular: calling %s (prompt=%d chars, max_tokens=%d)",
		d.config.Model.ID, len(conversationText), d.config.Model.MaxOutputTokens)
	return d.callLocalModel(ctx, distillSystemPrompt, distillUserPromptPrefix+conversationText)
}

// callLocalModel sends a single user prompt to the local model and returns the text response.
func (d *Distiller) callLocalModel(ctx context.Context, system, userPrompt string) (string, error) {
	llmCtx := &ai.Context{
		System: system,
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, userPrompt),
		},
	}

	maxTokens := d.config.Model.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = defaultLocalMaxTokens
	}

	stream := d.config.Provider.Stream(ctx, d.config.Model, llmCtx, &ai.StreamOptions{
		MaxTokens: maxTokens,
	})

	// Consume stream events, collect text from the final result.
	for range stream.Events() {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	result := stream.Result()
	if result == nil {
		return "", fmt.Errorf("local model stream completed without result")
	}

	text := extractText(result)
	if text == "" {
		return "", fmt.Errorf("local model returned empty response")
	}

	return text, nil
}

// formatMessages concatenates message text for the distillation prompt.
func formatMessages(msgs []ai.Message) string {
	var sb strings.Builder
	for _, msg := range msgs {
		role := string(msg.Role)
		text := extractMessageText(msg)
		if text != "" {
			fmt.Fprintf(&sb, "[%s]: %s\n\n", role, text)
		}
	}
	return sb.String()
}

// extractText pulls concatenated text from an AssistantMessage's content blocks.
func extractText(msg *ai.AssistantMessage) string {
	var parts []string
	for _, c := range msg.Content {
		if c.Type == ai.ContentText && c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// extractMessageText pulls displayable text from a Message.
func extractMessageText(msg ai.Message) string {
	var parts []string
	for _, c := range msg.Content {
		switch c.Type {
		case ai.ContentText:
			if c.Text != "" {
				parts = append(parts, c.Text)
			}
		case ai.ContentToolUse:
			parts = append(parts, fmt.Sprintf("[call:%s]", c.Name))
		case ai.ContentToolResult:
			parts = append(parts, fmt.Sprintf("[result:%s]", c.ResultText))
		}
	}
	return strings.Join(parts, "\n")
}
