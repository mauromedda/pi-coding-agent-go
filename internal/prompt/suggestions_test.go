// ABOUTME: Tests for AI-generated prompt suggestions parsing and formatting
// ABOUTME: Covers JSON parsing, display formatting, and prompt generation

package prompt

import (
	"strings"
	"testing"
)

func TestParseSuggestions_ValidJSON(t *testing.T) {
	input := `["How do I test this?", "Can you optimize?", "What about errors?"]`
	got := ParseSuggestions(input)

	if len(got) != 3 {
		t.Fatalf("ParseSuggestions valid JSON: got %d suggestions, want 3", len(got))
	}
	if got[0].Text != "How do I test this?" {
		t.Errorf("got[0].Text = %q; want %q", got[0].Text, "How do I test this?")
	}
	if got[1].Text != "Can you optimize?" {
		t.Errorf("got[1].Text = %q; want %q", got[1].Text, "Can you optimize?")
	}
	if got[2].Text != "What about errors?" {
		t.Errorf("got[2].Text = %q; want %q", got[2].Text, "What about errors?")
	}
}

func TestParseSuggestions_InvalidJSON(t *testing.T) {
	got := ParseSuggestions("not valid json at all")
	if len(got) != 0 {
		t.Errorf("ParseSuggestions invalid JSON: got %d suggestions, want 0", len(got))
	}
}

func TestParseSuggestions_EmptyArray(t *testing.T) {
	got := ParseSuggestions("[]")
	if len(got) != 0 {
		t.Errorf("ParseSuggestions empty array: got %d suggestions, want 0", len(got))
	}
}

func TestParseSuggestions_EmptyString(t *testing.T) {
	got := ParseSuggestions("")
	if len(got) != 0 {
		t.Errorf("ParseSuggestions empty string: got %d suggestions, want 0", len(got))
	}
}

func TestFormatSuggestions(t *testing.T) {
	suggestions := []Suggestion{
		{Text: "How do I test this?"},
		{Text: "Can you optimize?"},
		{Text: "What about errors?"},
	}
	got := FormatSuggestions(suggestions)

	want := "1. How do I test this?\n2. Can you optimize?\n3. What about errors?"
	if got != want {
		t.Errorf("FormatSuggestions:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestFormatSuggestions_Empty(t *testing.T) {
	got := FormatSuggestions(nil)
	if got != "" {
		t.Errorf("FormatSuggestions(nil) = %q; want empty string", got)
	}

	got = FormatSuggestions([]Suggestion{})
	if got != "" {
		t.Errorf("FormatSuggestions([]) = %q; want empty string", got)
	}
}

func TestSuggestionsPrompt(t *testing.T) {
	got := SuggestionsPrompt()
	if got == "" {
		t.Fatal("SuggestionsPrompt() returned empty string")
	}
	if !strings.Contains(got, "JSON") {
		t.Errorf("SuggestionsPrompt() should contain \"JSON\", got %q", got)
	}
}
