// ABOUTME: AI-generated follow-up prompt suggestions with JSON parsing
// ABOUTME: Parses LLM suggestion arrays and formats them for TUI display

package prompt

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Suggestion represents a follow-up prompt suggestion.
type Suggestion struct {
	Text string
}

// ParseSuggestions extracts suggestions from a JSON array string.
// Expected format: ["suggestion 1", "suggestion 2", "suggestion 3"]
// Returns an empty slice on parse error or empty input.
func ParseSuggestions(jsonStr string) []Suggestion {
	if jsonStr == "" {
		return nil
	}

	var raw []string
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil
	}

	suggestions := make([]Suggestion, 0, len(raw))
	for _, s := range raw {
		suggestions = append(suggestions, Suggestion{Text: s})
	}
	return suggestions
}

// FormatSuggestions formats suggestions as a numbered list for display.
// Returns an empty string when there are no suggestions.
func FormatSuggestions(suggestions []Suggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	var b strings.Builder
	for i, s := range suggestions {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d. %s", i+1, s.Text)
	}
	return b.String()
}

// SuggestionsPrompt returns the prompt to ask the LLM for follow-up suggestions.
func SuggestionsPrompt() string {
	return `Based on the conversation so far, suggest 2-3 brief follow-up questions or actions the user might want to take next. Return as a JSON array of strings, nothing else. Example: ["How do I test this?", "Can you optimize the performance?", "What about error handling?"]`
}
