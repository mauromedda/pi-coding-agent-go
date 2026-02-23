// ABOUTME: Intent classification types for routing user messages to appropriate agent modes.
// ABOUTME: Defines Intent enum, Classification result, and Signal contributing factors.

package intent

import "fmt"

// Intent represents the classified purpose of a user message.
type Intent int

const (
	IntentPlan      Intent = iota // Planning, architecture, design
	IntentExecute                 // Building, implementing, coding
	IntentExplore                 // Reading, searching, understanding
	IntentDebug                   // Fixing bugs, diagnosing issues
	IntentRefactor                // Restructuring, cleaning up
	IntentAmbiguous               // Cannot determine from heuristics alone
)

// String returns the human-readable name of the intent.
func (i Intent) String() string {
	switch i {
	case IntentPlan:
		return "plan"
	case IntentExecute:
		return "execute"
	case IntentExplore:
		return "explore"
	case IntentDebug:
		return "debug"
	case IntentRefactor:
		return "refactor"
	case IntentAmbiguous:
		return "ambiguous"
	default:
		return fmt.Sprintf("unknown(%d)", int(i))
	}
}

// Classification holds the result of intent classification.
type Classification struct {
	Intent     Intent
	Confidence float64  // 0.0-1.0
	Source     string   // "heuristic" or "model"
	Signals    []Signal // Contributing factors
}

// Signal represents a factor that contributed to classification.
type Signal struct {
	Name   string  // e.g., "keyword_match", "question_mark"
	Weight float64 // contribution to confidence
	Detail string  // e.g., the matched keyword
}
