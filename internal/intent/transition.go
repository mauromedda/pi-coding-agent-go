// ABOUTME: Mid-conversation mode transition detector.
// ABOUTME: Detects when user intent shifts and suggests mode changes with reason tracking.

package intent

import "fmt"

// transitionConfidenceThreshold is the minimum confidence required to trigger a transition.
const transitionConfidenceThreshold = 0.6

// TransitionDetector analyzes conversation flow to detect intent changes.
type TransitionDetector struct {
	currentIntent Intent
	history       []Intent
}

// NewTransitionDetector creates a detector starting with the given intent.
func NewTransitionDetector(initial Intent) *TransitionDetector {
	return &TransitionDetector{
		currentIntent: initial,
		history:       []Intent{initial},
	}
}

// Transition represents a detected mode change.
type Transition struct {
	From   Intent
	To     Intent
	Reason string
}

// Detect checks if the latest classification suggests a mode transition.
// Returns a Transition if a change is warranted, nil otherwise.
// Rules:
//  1. Same intent as current: no transition.
//  2. Ambiguous intent: no transition (avoid flapping).
//  3. Low confidence: no transition.
//  4. Otherwise: transition with descriptive reason.
func (d *TransitionDetector) Detect(latest Classification) *Transition {
	// No transition for same intent.
	if latest.Intent == d.currentIntent {
		return nil
	}

	// No transition for ambiguous classification.
	if latest.Intent == IntentAmbiguous {
		return nil
	}

	// No transition below confidence threshold.
	if latest.Confidence < transitionConfidenceThreshold {
		return nil
	}

	reason := fmt.Sprintf("%s -> %s (confidence: %.2f)", d.currentIntent, latest.Intent, latest.Confidence)

	tr := &Transition{
		From:   d.currentIntent,
		To:     latest.Intent,
		Reason: reason,
	}

	d.currentIntent = latest.Intent
	d.history = append(d.history, latest.Intent)

	return tr
}

// Current returns the current intent.
func (d *TransitionDetector) Current() Intent {
	return d.currentIntent
}

// History returns the intent history, starting with the initial intent.
func (d *TransitionDetector) History() []Intent {
	result := make([]Intent, len(d.history))
	copy(result, d.history)
	return result
}
