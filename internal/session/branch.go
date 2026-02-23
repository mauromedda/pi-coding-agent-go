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

// TreeNode represents a node in the session branch tree.
type TreeNode struct {
	ID       string
	ParentID string
	Summary  string
	Children []*TreeNode
}

// BuildTree constructs a tree from JSONL records for /tree command navigation.
// The root node is derived from the session_start record. Branch records are
// attached as children using their ParentID. Returns nil if no session_start
// record is found or records is empty.
func BuildTree(records []Record) *TreeNode {
	if len(records) == 0 {
		return nil
	}

	// Find root from session_start.
	var root *TreeNode
	nodesByID := make(map[string]*TreeNode)

	for _, rec := range records {
		switch rec.Type {
		case RecordSessionStart:
			var sd SessionStartData
			if err := rec.Unmarshal(&sd); err != nil {
				continue
			}
			root = &TreeNode{ID: sd.ID}
			nodesByID[sd.ID] = root

		case RecordBranch:
			var bd BranchData
			if err := rec.Unmarshal(&bd); err != nil {
				continue
			}
			child := &TreeNode{
				ID:       bd.Summary, // use summary as ID for branch nodes
				ParentID: bd.ParentID,
				Summary:  bd.Summary,
			}
			nodesByID[child.ID] = child

			if parent, ok := nodesByID[bd.ParentID]; ok {
				parent.Children = append(parent.Children, child)
			}
		}
	}

	return root
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
