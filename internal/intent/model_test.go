// ABOUTME: Tests for LLM-based intent classifier with canned responses.
// ABOUTME: Covers valid JSON, wrapped JSON, invalid input, all intent types, and error paths.

package intent

import (
	"errors"
	"testing"
)

func TestLLMClassifier_ValidJSON(t *testing.T) {
	t.Parallel()

	callLLM := func(_, _ string) (string, error) {
		return `{"intent": "execute", "confidence": 0.85}`, nil
	}

	c := NewLLMClassifier(callLLM)
	got, err := c.Classify("implement the handler")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Intent != IntentExecute {
		t.Errorf("Intent = %v; want %v", got.Intent, IntentExecute)
	}
	if got.Confidence != 0.85 {
		t.Errorf("Confidence = %.2f; want 0.85", got.Confidence)
	}
	if got.Source != "model" {
		t.Errorf("Source = %q; want %q", got.Source, "model")
	}
}

func TestLLMClassifier_WrappedJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		want     Intent
	}{
		{
			"text before JSON",
			`Here is my analysis: {"intent": "plan", "confidence": 0.9}`,
			IntentPlan,
		},
		{
			"text after JSON",
			`{"intent": "debug", "confidence": 0.8} That's my classification.`,
			IntentDebug,
		},
		{
			"newlines around JSON",
			"\n\n{\"intent\": \"refactor\", \"confidence\": 0.75}\n\n",
			IntentRefactor,
		},
		{
			"markdown code block",
			"```json\n{\"intent\": \"explore\", \"confidence\": 0.88}\n```",
			IntentExplore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			callLLM := func(_, _ string) (string, error) {
				return tt.response, nil
			}
			c := NewLLMClassifier(callLLM)
			got, err := c.Classify("test input")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Intent != tt.want {
				t.Errorf("Intent = %v; want %v", got.Intent, tt.want)
			}
		})
	}
}

func TestLLMClassifier_InvalidJSON(t *testing.T) {
	t.Parallel()

	callLLM := func(_, _ string) (string, error) {
		return "I think the intent is planning", nil
	}

	c := NewLLMClassifier(callLLM)
	_, err := c.Classify("test input")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestLLMClassifier_LLMError(t *testing.T) {
	t.Parallel()

	callLLM := func(_, _ string) (string, error) {
		return "", errors.New("connection refused")
	}

	c := NewLLMClassifier(callLLM)
	_, err := c.Classify("test input")
	if err == nil {
		t.Fatal("expected error when LLM call fails")
	}
}

func TestParseLLMResponse_AllIntents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		intentStr  string
		wantIntent Intent
	}{
		{"plan", "plan", IntentPlan},
		{"execute", "execute", IntentExecute},
		{"explore", "explore", IntentExplore},
		{"debug", "debug", IntentDebug},
		{"refactor", "refactor", IntentRefactor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := `{"intent": "` + tt.intentStr + `", "confidence": 0.9}`
			got, err := parseLLMResponse(response)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Intent != tt.wantIntent {
				t.Errorf("Intent = %v; want %v", got.Intent, tt.wantIntent)
			}
			if got.Source != "model" {
				t.Errorf("Source = %q; want %q", got.Source, "model")
			}
		})
	}
}

func TestParseLLMResponse_InvalidIntent(t *testing.T) {
	t.Parallel()

	_, err := parseLLMResponse(`{"intent": "dance", "confidence": 0.9}`)
	if err == nil {
		t.Fatal("expected error for unknown intent string")
	}
}

func TestParseLLMResponse_MissingFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
	}{
		{"missing intent", `{"confidence": 0.9}`},
		{"missing confidence", `{"intent": "plan"}`},
		{"empty object", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLLMResponse(tt.response)
			if err == nil {
				t.Fatal("expected error for missing fields")
			}
		})
	}
}
