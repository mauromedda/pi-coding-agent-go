// ABOUTME: LLM-based intent classifier using structured prompt for disambiguation.
// ABOUTME: Implements ModelClassifier interface; used as fallback when heuristics are ambiguous.

package intent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LLMClassifier classifies intent by sending a structured prompt to an LLM.
// It implements the ModelClassifier interface.
type LLMClassifier struct {
	callLLM func(systemPrompt, userPrompt string) (string, error)
}

// NewLLMClassifier creates a classifier that uses the provided LLM call function.
// The callLLM function should send the system+user prompts and return the response text.
func NewLLMClassifier(callLLM func(systemPrompt, userPrompt string) (string, error)) *LLMClassifier {
	return &LLMClassifier{callLLM: callLLM}
}

// Classify sends a classification prompt to the LLM and parses the response.
func (c *LLMClassifier) Classify(input string) (Classification, error) {
	system := `You are an intent classifier. Given a user message to a coding assistant, classify the intent.
Respond with ONLY a JSON object: {"intent": "<plan|execute|explore|debug|refactor>", "confidence": <0.0-1.0>}
Do not include any other text.`

	user := fmt.Sprintf("Classify this message's intent:\n\n%s", input)

	response, err := c.callLLM(system, user)
	if err != nil {
		return Classification{}, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseLLMResponse(response)
}

// llmResponse is the expected JSON structure from the LLM.
type llmResponse struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
}

// parseLLMResponse extracts intent and confidence from the LLM's JSON response.
// It handles JSON embedded in surrounding text or markdown code blocks.
func parseLLMResponse(response string) (Classification, error) {
	jsonStr, ok := extractJSON(response)
	if !ok {
		return Classification{}, fmt.Errorf("no JSON object found in response: %q", response)
	}

	var resp llmResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return Classification{}, fmt.Errorf("invalid JSON: %w", err)
	}

	if resp.Intent == "" {
		return Classification{}, fmt.Errorf("missing intent field in response")
	}
	if resp.Confidence == 0 {
		return Classification{}, fmt.Errorf("missing or zero confidence field in response")
	}

	intent, err := parseIntentString(resp.Intent)
	if err != nil {
		return Classification{}, err
	}

	return Classification{
		Intent:     intent,
		Confidence: resp.Confidence,
		Source:     "model",
		Signals: []Signal{{
			Name:   "llm_classification",
			Weight: resp.Confidence,
			Detail: resp.Intent,
		}},
	}, nil
}

// extractJSON finds the first JSON object in the string by locating matching braces.
func extractJSON(s string) (string, bool) {
	start := strings.Index(s, "{")
	if start == -1 {
		return "", false
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end <= start {
		return "", false
	}
	return s[start : end+1], true
}

// parseIntentString maps a string to the Intent enum.
func parseIntentString(s string) (Intent, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "plan":
		return IntentPlan, nil
	case "execute":
		return IntentExecute, nil
	case "explore":
		return IntentExplore, nil
	case "debug":
		return IntentDebug, nil
	case "refactor":
		return IntentRefactor, nil
	default:
		return IntentAmbiguous, fmt.Errorf("unknown intent: %q", s)
	}
}
