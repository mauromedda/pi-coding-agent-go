// ABOUTME: Orchestrator: heuristic-first classification with optional LLM fallback.
// ABOUTME: Auto-escalates to plan mode when input implies broad multi-file scope.

package intent

import (
	"regexp"
	"strings"
)

// ClassifierConfig holds configuration for the intent classifier.
type ClassifierConfig struct {
	HeuristicThreshold float64         // Min confidence to accept heuristic result (default 0.7).
	AutoPlanFileCount  int             // Auto-escalate to Plan if estimated files > this (default 5).
	LLMFallback        ModelClassifier // Optional; if nil, returns heuristic result even if ambiguous.
}

// ModelClassifier is the interface for LLM-based classification.
type ModelClassifier interface {
	Classify(input string) (Classification, error)
}

// Classifier orchestrates intent classification with heuristic-first strategy.
type Classifier struct {
	config ClassifierConfig
}

// NewClassifier creates a classifier with the given config, applying defaults.
func NewClassifier(cfg ClassifierConfig) *Classifier {
	if cfg.HeuristicThreshold == 0 {
		cfg.HeuristicThreshold = 0.7
	}
	if cfg.AutoPlanFileCount == 0 {
		cfg.AutoPlanFileCount = 5
	}
	return &Classifier{config: cfg}
}

// scopePatterns detect inputs that imply broad, multi-file changes.
var scopePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bacross\s+(all\s+)?files?\b`),
	regexp.MustCompile(`(?i)\bentire\s+codebase\b`),
	regexp.MustCompile(`(?i)\bproject[- ]wide\b`),
	regexp.MustCompile(`(?i)\beverywhere\b`),
	regexp.MustCompile(`(?i)\bevery\s+file\b`),
	regexp.MustCompile(`(?i)\ball\s+files?\b`),
}

// Classify determines the intent of a user message.
// Strategy: heuristic first. If confidence >= threshold, return immediately.
// If ambiguous and LLM fallback is configured, delegate to model.
// If input implies large scope, escalate to Plan.
func (c *Classifier) Classify(input string) (Classification, error) {
	result := ClassifyHeuristic(input)

	// Auto-escalation: if input implies broad multi-file scope, force Plan.
	if c.shouldAutoEscalate(input) {
		return Classification{
			Intent:     IntentPlan,
			Confidence: 0.9,
			Source:     "auto-escalation",
			Signals: append(result.Signals, Signal{
				Name:   "auto_escalation",
				Weight: 1.0,
				Detail: "scope exceeds threshold",
			}),
		}, nil
	}

	// If heuristic is confident enough and not ambiguous, return.
	if result.Confidence >= c.config.HeuristicThreshold && result.Intent != IntentAmbiguous {
		return result, nil
	}

	// LLM fallback if available.
	if c.config.LLMFallback != nil {
		modelResult, err := c.config.LLMFallback.Classify(input)
		if err != nil {
			// Fall back to heuristic on LLM error.
			return result, nil
		}
		return modelResult, nil
	}

	return result, nil
}

// shouldAutoEscalate checks if the input implies a scope that warrants plan mode.
func (c *Classifier) shouldAutoEscalate(input string) bool {
	lower := strings.ToLower(input)
	for _, p := range scopePatterns {
		if p.MatchString(lower) {
			return true
		}
	}
	return false
}
