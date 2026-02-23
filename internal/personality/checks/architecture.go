// ABOUTME: Architecture check for dependency direction, layer violations, and interface compliance
// ABOUTME: Enforces structural constraints based on codebase organization

package checks

// ArchitectureCheck analyzes architectural concerns at configurable rigor levels.
type ArchitectureCheck struct {
	level string
}

// NewArchitectureCheck creates an ArchitectureCheck at the given level.
func NewArchitectureCheck(level string) *ArchitectureCheck {
	return &ArchitectureCheck{level: level}
}

// Name returns the check name.
func (c *ArchitectureCheck) Name() string { return "architecture" }

// Analyze runs architecture analysis and returns instructions based on level and context.
func (c *ArchitectureCheck) Analyze(ctx CheckContext) CheckResult {
	r := CheckResult{
		Name:  "architecture",
		Level: c.level,
		Score: 100,
	}

	// Minimal: basic structure
	r.Instructions = append(r.Instructions, "Keep functions focused on a single responsibility")
	r.Score = 80

	if c.level == "minimal" {
		return r
	}

	// Standard: + dependency direction, package boundaries
	r.Instructions = append(r.Instructions,
		"Enforce dependency direction: inner packages must not import outer",
		"Respect package boundaries; avoid circular dependencies",
	)
	r.Score = 70

	if c.level == "standard" {
		return r
	}

	// Strict: + interface compliance, composition, layer separation
	r.Instructions = append(r.Instructions,
		"Define small interfaces (1-3 methods) at consumer side",
		"Prefer composition over inheritance; embed by value",
		"Separate domain logic from infrastructure concerns",
		"Verify no layer violations: handlers must not access repositories directly",
	)
	r.Score = 50

	if len(ctx.Languages) > 1 {
		r.Warnings = append(r.Warnings, "Multi-language codebase: verify consistent patterns across languages")
	}

	if c.level == "strict" {
		return r
	}

	// Paranoid: + ADR tracking, formal contracts, boundary tests
	r.Instructions = append(r.Instructions,
		"Document architectural decisions in ADR format",
		"Define formal contracts between service boundaries",
		"Write boundary integration tests for all module interfaces",
		"Review for hexagonal architecture compliance",
	)
	r.Score = 40

	return r
}
