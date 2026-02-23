// ABOUTME: Context compaction: token-budget triggers, cut-point logic, LLM-based summarization
// ABOUTME: Reduces context size when approaching model token limits; injectable summarizer

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// CompactionConfig controls when and how compaction occurs.
type CompactionConfig struct {
	ReserveTokens    int // tokens reserved for response generation
	KeepRecentTokens int // tokens worth of recent messages to preserve
}

// CompactResult holds the output of a compaction operation.
type CompactResult struct {
	Messages       []ai.Message    // compacted message list (summary + ack + kept)
	Summary        string          // the generated summary text
	FileOps        CompactionEntry // cumulative file tracking
	TokensBefore   int             // token count before compaction
	FirstKeptIndex int             // index into original messages where kept portion starts
}

// SummarizerFunc is an injectable function that produces a summary from messages.
type SummarizerFunc func(ctx context.Context, messages []ai.Message, previousSummary string) (string, error)

// CompactionEntry records metadata about a compacted message span.
type CompactionEntry struct {
	FilesRead    []string // file paths that were read during the span
	FilesWritten []string // file paths that were written/edited during the span
	MessageCount int      // number of messages in the compacted span
}

// readTools are tool names that read files.
var readTools = map[string]bool{
	"Read": true, "Glob": true, "Grep": true,
}

// writeTools are tool names that write/edit files.
var writeTools = map[string]bool{
	"Write": true, "Edit": true, "NotebookEdit": true,
}

// ExtractFileOps scans messages for tool_use content blocks and extracts
// file paths categorized as read or written.
func ExtractFileOps(messages []ai.Message) CompactionEntry {
	entry := CompactionEntry{
		MessageCount: len(messages),
	}

	readSeen := make(map[string]bool)
	writeSeen := make(map[string]bool)

	for _, msg := range messages {
		for _, c := range msg.Content {
			if c.Type != ai.ContentToolUse || len(c.Input) == 0 {
				continue
			}

			filePath := extractFilePath(c.Input)
			if filePath == "" {
				continue
			}

			if readTools[c.Name] {
				if !readSeen[filePath] {
					readSeen[filePath] = true
					entry.FilesRead = append(entry.FilesRead, filePath)
				}
			}
			if writeTools[c.Name] {
				if !writeSeen[filePath] {
					writeSeen[filePath] = true
					entry.FilesWritten = append(entry.FilesWritten, filePath)
				}
			}
		}
	}

	return entry
}

// extractFilePath pulls the file_path field from tool input JSON.
func extractFilePath(input json.RawMessage) string {
	var args struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return ""
	}
	if args.FilePath != "" {
		return args.FilePath
	}
	return args.NotebookPath
}

// ShouldCompact returns true when the estimated token count of messages
// exceeds the available budget (contextWindow - reserveTokens).
func ShouldCompact(messages []ai.Message, contextWindow int, cfg CompactionConfig) bool {
	tokens := EstimateMessagesTokens(messages)
	budget := contextWindow - cfg.ReserveTokens
	return tokens > budget
}

// FindCutPoint walks backward from the end of messages, accumulating tokens
// until keepRecentTokens is reached. Returns the index where older messages
// should be cut. Never cuts between a tool_use and its tool_result.
func FindCutPoint(messages []ai.Message, keepRecentTokens int) int {
	if len(messages) <= 1 {
		return 0
	}

	accumulated := 0
	cutIdx := len(messages) // start past the end

	for i := len(messages) - 1; i >= 0; i-- {
		accumulated += EstimateMessageTokens(messages[i])
		if accumulated >= keepRecentTokens {
			cutIdx = i
			break
		}
	}

	// If we never hit the budget, cut at 0 (keep everything)
	if cutIdx >= len(messages) {
		return 0
	}

	// Adjust forward to avoid cutting between tool_use and tool_result.
	// A tool_result at index i always pairs with the tool_use at index i-1.
	for cutIdx < len(messages) {
		if hasToolResult(messages[cutIdx]) && cutIdx > 0 && hasToolUse(messages[cutIdx-1]) {
			cutIdx++ // move cut past the tool_result
		} else {
			break
		}
	}

	// Don't cut past everything
	if cutIdx >= len(messages) {
		return len(messages) - 1
	}

	return cutIdx
}

// hasToolUse returns true if any content block in the message is a tool_use.
func hasToolUse(msg ai.Message) bool {
	for _, c := range msg.Content {
		if c.Type == ai.ContentToolUse {
			return true
		}
	}
	return false
}

// hasToolResult returns true if any content block in the message is a tool_result.
func hasToolResult(msg ai.Message) bool {
	for _, c := range msg.Content {
		if c.Type == ai.ContentToolResult {
			return true
		}
	}
	return false
}

// CompactWithLLM performs compaction using an injected summarizer function.
// It finds a cut point, summarizes the older messages, and returns a new
// message list with summary + acknowledgment + kept recent messages.
func CompactWithLLM(ctx context.Context, messages []ai.Message, cfg CompactionConfig, summarize SummarizerFunc) (*CompactResult, error) {
	tokensBefore := EstimateMessagesTokens(messages)
	cutIdx := FindCutPoint(messages, cfg.KeepRecentTokens)

	if cutIdx == 0 {
		// Nothing to compact
		return &CompactResult{
			Messages:       messages,
			TokensBefore:   tokensBefore,
			FirstKeptIndex: 0,
		}, nil
	}

	oldMessages := messages[:cutIdx]
	recentMessages := messages[cutIdx:]

	// Extract file ops from the compacted span
	fileOps := ExtractFileOps(oldMessages)

	// Call the injected summarizer
	summary, err := summarize(ctx, oldMessages, "")
	if err != nil {
		return nil, fmt.Errorf("compaction summarizer: %w", err)
	}

	// Build file tracking tags
	var fileTags strings.Builder
	if len(fileOps.FilesRead) > 0 {
		fileTags.WriteString("\n\n<read-files>\n")
		for _, f := range fileOps.FilesRead {
			fileTags.WriteString("- " + f + "\n")
		}
		fileTags.WriteString("</read-files>")
	}
	if len(fileOps.FilesWritten) > 0 {
		fileTags.WriteString("\n\n<modified-files>\n")
		for _, f := range fileOps.FilesWritten {
			fileTags.WriteString("- " + f + "\n")
		}
		fileTags.WriteString("</modified-files>")
	}

	summaryText := fmt.Sprintf("[Context Summary]\n%s%s\n[End Summary]", summary, fileTags.String())

	// Build compacted message list
	compacted := make([]ai.Message, 0, len(recentMessages)+2)
	compacted = append(compacted, ai.NewTextMessage(ai.RoleUser, summaryText))
	compacted = append(compacted, ai.NewTextMessage(ai.RoleAssistant,
		"I understand the context. Let me continue from where we left off."))
	compacted = append(compacted, recentMessages...)

	return &CompactResult{
		Messages:       compacted,
		Summary:        summary,
		FileOps:        fileOps,
		TokensBefore:   tokensBefore,
		FirstKeptIndex: cutIdx,
	}, nil
}
