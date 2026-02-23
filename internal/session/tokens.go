// ABOUTME: Token estimation heuristics for context budget management
// ABOUTME: Chars ÷ 4 approximation; sums across content blocks and messages

package session

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// EstimateTokens returns an approximate token count for a text string.
// Uses the chars ÷ 4 heuristic which is accurate within ~10% for English text.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4 // ceiling division
}

// EstimateContentTokens estimates tokens for a single content block.
func EstimateContentTokens(c ai.Content) int {
	switch c.Type {
	case ai.ContentText:
		return EstimateTokens(c.Text)
	case ai.ContentThinking:
		return EstimateTokens(c.Thinking)
	case ai.ContentToolUse:
		// Tool name + JSON input
		return EstimateTokens(c.Name) + EstimateTokens(string(c.Input))
	case ai.ContentToolResult:
		return EstimateTokens(c.ResultText)
	case ai.ContentImage:
		// Images are roughly 1000 tokens regardless of size
		return 1000
	default:
		return 0
	}
}

// EstimateMessageTokens estimates tokens for a single message.
func EstimateMessageTokens(msg ai.Message) int {
	tokens := 4 // overhead per message (role, separators)
	for _, c := range msg.Content {
		tokens += EstimateContentTokens(c)
	}
	return tokens
}

// EstimateMessagesTokens estimates tokens for a slice of messages.
func EstimateMessagesTokens(msgs []ai.Message) int {
	total := 0
	for _, msg := range msgs {
		total += EstimateMessageTokens(msg)
	}
	return total
}
