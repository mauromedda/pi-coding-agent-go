// ABOUTME: Check interface and context for personality verification layers
// ABOUTME: Each check analyzes agent behavior and generates prompt instructions

package checks

// CheckContext provides information about the current agent state for checks to analyze.
type CheckContext struct {
	FilesChanged          int
	LinesChanged          int
	HasTests              bool
	HasErrorHandling      bool
	Languages             []string
	IsSecurityRelated     bool
	IsPerformanceCritical bool
}

// CheckResult holds the output of a check analysis.
type CheckResult struct {
	Name         string   // Check name
	Level        string   // Active level
	Instructions []string // Prompt instructions to inject
	Warnings     []string // Issues detected
	Score        int      // 0-100 quality score
}

// Check is the interface that all personality checks implement.
type Check interface {
	Name() string
	Analyze(ctx CheckContext) CheckResult
}

// NewCheck creates a check by name and level.
// Returns nil if the check name is unknown.
func NewCheck(name, level string) Check {
	switch name {
	case "security":
		return NewSecurityCheck(level)
	case "performance":
		return NewPerformanceCheck(level)
	case "quality":
		return NewQualityCheck(level)
	case "architecture":
		return NewArchitectureCheck(level)
	case "factual":
		return NewFactualCheck(level)
	default:
		return nil
	}
}

// AllCheckNames returns the list of known check names.
func AllCheckNames() []string {
	return []string{"security", "performance", "quality", "architecture", "factual"}
}
