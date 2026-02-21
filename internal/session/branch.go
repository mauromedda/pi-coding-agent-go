// ABOUTME: Branch summarization for session tree navigation
// ABOUTME: Creates summary records when branching off from a conversation point

package session

import (
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// BranchData holds branch metadata for JSONL persistence.
type BranchData struct {
	ParentID string `json:"parent_id"`
	Summary  string `json:"summary"`
}

// CreateBranch records a branch point in the session.
func (s *Session) CreateBranch(parentToolID, summary string) error {
	return s.Writer.WriteRecord(RecordBranch, BranchData{
		ParentID: parentToolID,
		Summary:  summary,
	})
}

// SummarizeBranch creates a brief summary of a message sequence.
func SummarizeBranch(messages []ai.Message) string {
	if len(messages) == 0 {
		return ""
	}

	// Use the first user message as the branch summary
	for _, msg := range messages {
		if msg.Role == ai.RoleUser {
			for _, c := range msg.Content {
				if c.Type == ai.ContentText {
					text := c.Text
					if len(text) > 100 {
						text = text[:100] + "..."
					}
					return fmt.Sprintf("User asked: %s", text)
				}
			}
		}
	}
	return fmt.Sprintf("Branch with %d messages", len(messages))
}
