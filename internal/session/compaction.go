// ABOUTME: Context compaction: summarize old messages, keep recent ones
// ABOUTME: Reduces context size when approaching model token limits

package session

import (
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

const keepRecentMessages = 10

// Compact summarizes older messages into a single summary message,
// keeping the most recent messages intact.
func Compact(messages []ai.Message) ([]ai.Message, string, error) {
	if len(messages) <= keepRecentMessages {
		return messages, "", nil
	}

	// Split into old and recent
	oldMessages := messages[:len(messages)-keepRecentMessages]
	recentMessages := messages[len(messages)-keepRecentMessages:]

	// Build summary from old messages
	summary := buildSummary(oldMessages)

	// Create compacted message list
	compacted := make([]ai.Message, 0, keepRecentMessages+1)
	compacted = append(compacted, ai.NewTextMessage(ai.RoleUser,
		fmt.Sprintf("[Context Summary]\n%s\n[End Summary]", summary)))
	compacted = append(compacted, ai.NewTextMessage(ai.RoleAssistant,
		"I understand the context. Let me continue from where we left off."))
	compacted = append(compacted, recentMessages...)

	return compacted, summary, nil
}

// buildSummary creates a text summary from messages.
func buildSummary(messages []ai.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(string(msg.Role))
		b.WriteString(": ")
		for _, c := range msg.Content {
			if c.Type == ai.ContentText {
				text := c.Text
				if len(text) > 200 {
					text = text[:200] + "..."
				}
				b.WriteString(text)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
