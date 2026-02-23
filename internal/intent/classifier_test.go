// ABOUTME: Tests for the orchestrator classifier with heuristic-first strategy and LLM fallback.
// ABOUTME: Covers threshold acceptance, fallback delegation, error recovery, and auto-escalation.

package intent

import (
	"errors"
	"testing"
)

// mockModelClassifier implements ModelClassifier for testing.
type mockModelClassifier struct {
	result Classification
	err    error
	called bool
}

func (m *mockModelClassifier) Classify(_ string) (Classification, error) {
	m.called = true
	return m.result, m.err
}

func TestClassifier_HeuristicAccepted(t *testing.T) {
	t.Parallel()

	// "implement the HTTP handler" gives a clear execute intent with high confidence.
	c := NewClassifier(ClassifierConfig{
		HeuristicThreshold: 0.7,
	})

	got, err := c.Classify("implement the HTTP handler for users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Intent != IntentExecute {
		t.Errorf("Intent = %v; want %v", got.Intent, IntentExecute)
	}
	if got.Confidence < 0.7 {
		t.Errorf("Confidence = %.2f; want >= 0.7", got.Confidence)
	}
}

func TestClassifier_HeuristicBelowThreshold_NoFallback(t *testing.T) {
	t.Parallel()

	// With threshold at 0.99, most heuristic results won't meet it.
	// Without LLM fallback, should return the heuristic result anyway.
	c := NewClassifier(ClassifierConfig{
		HeuristicThreshold: 0.99,
	})

	got, err := c.Classify("plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still return something (heuristic result), not an error.
	if got.Intent != IntentPlan {
		// With a single keyword "plan", confidence is below 0.99 but
		// it should still return the heuristic result when no fallback.
		t.Errorf("Intent = %v; want %v", got.Intent, IntentPlan)
	}
}

func TestClassifier_HeuristicBelowThreshold_WithFallback(t *testing.T) {
	t.Parallel()

	mock := &mockModelClassifier{
		result: Classification{
			Intent:     IntentPlan,
			Confidence: 0.95,
			Source:     "model",
		},
	}

	c := NewClassifier(ClassifierConfig{
		HeuristicThreshold: 0.99,
		LLMFallback:        mock,
	})

	got, err := c.Classify("plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.called {
		t.Error("expected LLM fallback to be called")
	}
	if got.Intent != IntentPlan {
		t.Errorf("Intent = %v; want %v", got.Intent, IntentPlan)
	}
	if got.Source != "model" {
		t.Errorf("Source = %q; want %q", got.Source, "model")
	}
}

func TestClassifier_LLMFallbackError(t *testing.T) {
	t.Parallel()

	mock := &mockModelClassifier{
		err: errors.New("LLM unavailable"),
	}

	c := NewClassifier(ClassifierConfig{
		HeuristicThreshold: 0.99,
		LLMFallback:        mock,
	})

	// Should fall back to heuristic result on LLM error.
	got, err := c.Classify("plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.called {
		t.Error("expected LLM fallback to be called")
	}
	// Should return heuristic result, not an error.
	if got.Source != "heuristic" {
		t.Errorf("Source = %q; want %q (heuristic fallback)", got.Source, "heuristic")
	}
}

func TestClassifier_AutoEscalation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"across all files", "implement logging across all files in the project"},
		{"entire codebase", "add error handling to the entire codebase"},
		{"project-wide", "apply project-wide formatting changes"},
		{"everywhere", "update the API version everywhere"},
		{"every file", "add copyright headers to every file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewClassifier(ClassifierConfig{})
			got, err := c.Classify(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Intent != IntentPlan {
				t.Errorf("Intent = %v; want %v (auto-escalation)", got.Intent, IntentPlan)
			}
			if got.Source != "auto-escalation" {
				t.Errorf("Source = %q; want %q", got.Source, "auto-escalation")
			}
		})
	}
}

func TestClassifier_NoAutoEscalation_NarrowScope(t *testing.T) {
	t.Parallel()

	// A simple execute intent should NOT trigger auto-escalation.
	c := NewClassifier(ClassifierConfig{})
	got, err := c.Classify("implement the HTTP handler for users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source == "auto-escalation" {
		t.Errorf("did not expect auto-escalation for narrow-scope input")
	}
}

func TestClassifier_DefaultConfig(t *testing.T) {
	t.Parallel()

	c := NewClassifier(ClassifierConfig{})
	if c.config.HeuristicThreshold != 0.7 {
		t.Errorf("default HeuristicThreshold = %.2f; want 0.70", c.config.HeuristicThreshold)
	}
	if c.config.AutoPlanFileCount != 5 {
		t.Errorf("default AutoPlanFileCount = %d; want 5", c.config.AutoPlanFileCount)
	}
}

func TestClassifier_CustomConfig(t *testing.T) {
	t.Parallel()

	c := NewClassifier(ClassifierConfig{
		HeuristicThreshold: 0.5,
		AutoPlanFileCount:  10,
	})
	if c.config.HeuristicThreshold != 0.5 {
		t.Errorf("HeuristicThreshold = %.2f; want 0.50", c.config.HeuristicThreshold)
	}
	if c.config.AutoPlanFileCount != 10 {
		t.Errorf("AutoPlanFileCount = %d; want 10", c.config.AutoPlanFileCount)
	}
}

func TestClassifier_AmbiguousInput_WithFallback(t *testing.T) {
	t.Parallel()

	mock := &mockModelClassifier{
		result: Classification{
			Intent:     IntentExplore,
			Confidence: 0.8,
			Source:     "model",
		},
	}

	c := NewClassifier(ClassifierConfig{
		LLMFallback: mock,
	})

	// "hello" is ambiguous in heuristics.
	got, err := c.Classify("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.called {
		t.Error("expected LLM fallback to be called for ambiguous input")
	}
	if got.Intent != IntentExplore {
		t.Errorf("Intent = %v; want %v", got.Intent, IntentExplore)
	}
}
