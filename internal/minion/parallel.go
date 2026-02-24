// ABOUTME: Distributor implements the plural minion protocol (parallel extraction)
// ABOUTME: Splits context into chunks, processes each via local model in parallel, aggregates results

package minion

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

const defaultMaxWorkers = 3

// DistributorConfig holds the configuration for a Distributor.
type DistributorConfig struct {
	Provider   ai.ApiProvider
	Model      *ai.Model
	MaxWorkers int // max parallel local model calls (default 3)
	KeepRecent int // number of recent messages to pass through unmodified (default 4)
}

// Distributor implements the plural minion protocol: splits messages into chunks,
// processes each in parallel via a local model, and aggregates structured extracts.
type Distributor struct {
	config DistributorConfig
}

// NewDistributor creates a Distributor with the given configuration.
func NewDistributor(cfg DistributorConfig) *Distributor {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = defaultMaxWorkers
	}
	if cfg.KeepRecent <= 0 {
		cfg.KeepRecent = defaultKeepRecent
	}
	return &Distributor{config: cfg}
}

// Distribute splits older messages into chunks, processes each in parallel via
// the local model, and returns an aggregated summary + recent messages.
// If there are fewer messages than KeepRecent, returns the original messages unchanged.
func (d *Distributor) Distribute(ctx context.Context, msgs []ai.Message) ([]ai.Message, error) {
	if len(msgs) <= d.config.KeepRecent {
		return msgs, nil
	}

	olderMsgs := msgs[:len(msgs)-d.config.KeepRecent]
	recentMsgs := msgs[len(msgs)-d.config.KeepRecent:]

	chunks := splitIntoChunks(olderMsgs)
	if len(chunks) == 0 {
		return msgs, nil
	}

	extracts, err := d.processChunksParallel(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("distribute: %w", err)
	}

	aggregated := aggregateExtracts(extracts)

	summaryContent := ai.Content{Type: ai.ContentText, Text: "[Aggregated Context]\n" + aggregated}
	return prependSummary(summaryContent, recentMsgs), nil
}

// splitIntoChunks splits messages at natural boundaries: after tool results,
// or after assistant messages that don't contain tool calls (text-only responses).
func splitIntoChunks(msgs []ai.Message) [][]ai.Message {
	if len(msgs) == 0 {
		return nil
	}

	var chunks [][]ai.Message
	var current []ai.Message

	for _, m := range msgs {
		current = append(current, m)
		switch {
		case hasToolResultContent(m):
			chunks = append(chunks, current)
			current = nil
		case m.Role == ai.RoleAssistant && !hasToolCallContent(m):
			chunks = append(chunks, current)
			current = nil
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

// hasToolCallContent checks if a message contains any tool_use content blocks.
func hasToolCallContent(msg ai.Message) bool {
	for _, c := range msg.Content {
		if c.Type == ai.ContentToolUse {
			return true
		}
	}
	return false
}

// hasToolResultContent checks if a message contains any tool_result content blocks.
func hasToolResultContent(msg ai.Message) bool {
	for _, c := range msg.Content {
		if c.Type == ai.ContentToolResult {
			return true
		}
	}
	return false
}

// processChunksParallel sends each chunk to the local model concurrently,
// capped at MaxWorkers.
func (d *Distributor) processChunksParallel(ctx context.Context, chunks [][]ai.Message) ([]string, error) {
	extracts := make([]string, len(chunks))
	errs := make([]error, len(chunks))

	sem := make(chan struct{}, d.config.MaxWorkers)
	var wg sync.WaitGroup

	for i, chunk := range chunks {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(idx int, chunk []ai.Message) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs[idx] = ctx.Err()
				return
			}

			extract, err := d.extractFromChunk(ctx, chunk)
			if err != nil {
				errs[idx] = err
				return
			}
			extracts[idx] = extract
		}(i, chunk)
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return extracts, nil
}

// extractFromChunk sends a single chunk to the local model for structured extraction.
func (d *Distributor) extractFromChunk(ctx context.Context, chunk []ai.Message) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	chunkText := formatMessages(chunk)

	llmCtx := &ai.Context{
		System: extractSystemPrompt,
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, extractUserPromptPrefix+chunkText),
		},
	}

	maxTokens := d.config.Model.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = defaultLocalMaxTokens
	}

	stream := d.config.Provider.Stream(ctx, d.config.Model, llmCtx, &ai.StreamOptions{
		MaxTokens: maxTokens,
	})

	for range stream.Events() {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	result := stream.Result()
	if result == nil {
		return "", fmt.Errorf("local model stream completed without result")
	}

	return extractText(result), nil
}

// aggregateExtracts combines multiple chunk extractions into a single summary.
func aggregateExtracts(extracts []string) string {
	var sb strings.Builder
	for i, ext := range extracts {
		if ext == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n---\n")
		}
		fmt.Fprintf(&sb, "Chunk %d:\n%s", i+1, ext)
	}
	return sb.String()
}
