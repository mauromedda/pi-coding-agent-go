// ABOUTME: Tests for mid-conversation intent transition detection.
// ABOUTME: Covers same-intent no-op, mode switches, low-confidence rejection, and history tracking.

package intent

import "testing"

func TestTransitionDetector_NoTransition_SameIntent(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentExecute)
	tr := d.Detect(Classification{
		Intent:     IntentExecute,
		Confidence: 0.9,
		Source:     "heuristic",
	})
	if tr != nil {
		t.Errorf("expected nil transition for same intent; got %+v", tr)
	}
}

func TestTransitionDetector_PlanToExecute(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentPlan)
	tr := d.Detect(Classification{
		Intent:     IntentExecute,
		Confidence: 0.85,
		Source:     "heuristic",
	})
	if tr == nil {
		t.Fatal("expected transition from Plan to Execute")
	}
	if tr.From != IntentPlan {
		t.Errorf("From = %v; want %v", tr.From, IntentPlan)
	}
	if tr.To != IntentExecute {
		t.Errorf("To = %v; want %v", tr.To, IntentExecute)
	}
	if tr.Reason == "" {
		t.Error("expected non-empty Reason")
	}
}

func TestTransitionDetector_ExecuteToDebug(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentExecute)
	tr := d.Detect(Classification{
		Intent:     IntentDebug,
		Confidence: 0.85,
		Source:     "heuristic",
	})
	if tr == nil {
		t.Fatal("expected transition from Execute to Debug")
	}
	if tr.From != IntentExecute {
		t.Errorf("From = %v; want %v", tr.From, IntentExecute)
	}
	if tr.To != IntentDebug {
		t.Errorf("To = %v; want %v", tr.To, IntentDebug)
	}
}

func TestTransitionDetector_AnyToPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial Intent
	}{
		{"execute to plan", IntentExecute},
		{"explore to plan", IntentExplore},
		{"debug to plan", IntentDebug},
		{"refactor to plan", IntentRefactor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := NewTransitionDetector(tt.initial)
			tr := d.Detect(Classification{
				Intent:     IntentPlan,
				Confidence: 0.85,
				Source:     "heuristic",
			})
			if tr == nil {
				t.Fatalf("expected transition from %v to Plan", tt.initial)
			}
			if tr.To != IntentPlan {
				t.Errorf("To = %v; want %v", tr.To, IntentPlan)
			}
		})
	}
}

func TestTransitionDetector_LowConfidence_NoTransition(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentExecute)
	tr := d.Detect(Classification{
		Intent:     IntentDebug,
		Confidence: 0.3,
		Source:     "heuristic",
	})
	if tr != nil {
		t.Errorf("expected nil transition for low confidence; got %+v", tr)
	}
}

func TestTransitionDetector_AmbiguousIntent_NoTransition(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentExecute)
	tr := d.Detect(Classification{
		Intent:     IntentAmbiguous,
		Confidence: 0.9,
		Source:     "heuristic",
	})
	if tr != nil {
		t.Errorf("expected nil transition for ambiguous intent; got %+v", tr)
	}
}

func TestTransitionDetector_History(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentPlan)

	// Transition to Execute.
	d.Detect(Classification{
		Intent:     IntentExecute,
		Confidence: 0.9,
		Source:     "heuristic",
	})

	// Transition to Debug.
	d.Detect(Classification{
		Intent:     IntentDebug,
		Confidence: 0.9,
		Source:     "heuristic",
	})

	hist := d.History()
	if len(hist) != 3 {
		t.Fatalf("History length = %d; want 3", len(hist))
	}
	if hist[0] != IntentPlan {
		t.Errorf("History[0] = %v; want %v", hist[0], IntentPlan)
	}
	if hist[1] != IntentExecute {
		t.Errorf("History[1] = %v; want %v", hist[1], IntentExecute)
	}
	if hist[2] != IntentDebug {
		t.Errorf("History[2] = %v; want %v", hist[2], IntentDebug)
	}
}

func TestTransitionDetector_Current(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentExplore)
	if d.Current() != IntentExplore {
		t.Errorf("Current() = %v; want %v", d.Current(), IntentExplore)
	}

	// After a transition, Current should update.
	d.Detect(Classification{
		Intent:     IntentDebug,
		Confidence: 0.9,
		Source:     "heuristic",
	})
	if d.Current() != IntentDebug {
		t.Errorf("Current() after transition = %v; want %v", d.Current(), IntentDebug)
	}
}

func TestTransitionDetector_HistoryPreservedOnNoTransition(t *testing.T) {
	t.Parallel()

	d := NewTransitionDetector(IntentExecute)

	// Same intent; no transition.
	d.Detect(Classification{
		Intent:     IntentExecute,
		Confidence: 0.9,
		Source:     "heuristic",
	})

	hist := d.History()
	// Only the initial entry; no duplicate.
	if len(hist) != 1 {
		t.Errorf("History length = %d; want 1 (no transition occurred)", len(hist))
	}
}
