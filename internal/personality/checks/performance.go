// ABOUTME: Performance check analyzing complexity, concurrency, and memory patterns
// ABOUTME: Generates optimization instructions based on code characteristics

package checks

// PerformanceCheck analyzes performance concerns at configurable rigor levels.
type PerformanceCheck struct {
	level string
}

// NewPerformanceCheck creates a PerformanceCheck at the given level.
func NewPerformanceCheck(level string) *PerformanceCheck {
	return &PerformanceCheck{level: level}
}

// Name returns the check name.
func (c *PerformanceCheck) Name() string { return "performance" }

// Analyze runs performance analysis and returns instructions based on level and context.
func (c *PerformanceCheck) Analyze(ctx CheckContext) CheckResult {
	r := CheckResult{
		Name:  "performance",
		Level: c.level,
		Score: 100,
	}

	// Minimal: basic N+1 reminder
	r.Instructions = append(r.Instructions, "Avoid N+1 query patterns in data access code")
	r.Score = 80

	if c.level == "minimal" {
		return r
	}

	// Standard: + pre-allocation, connection pooling, query optimization
	r.Instructions = append(r.Instructions,
		"Use pre-allocation for slices and maps with known sizes",
		"Ensure connection pooling for database and HTTP clients",
		"Optimize queries: use indexes, avoid SELECT *",
	)
	r.Score = 70

	if ctx.IsPerformanceCritical {
		r.Warnings = append(r.Warnings, "Performance-critical code path detected; review carefully")
	}

	if c.level == "standard" {
		return r
	}

	// Strict: + profiling, benchmarking, memory budget
	r.Instructions = append(r.Instructions,
		"Add profiling hooks for hot paths",
		"Write benchmarks for performance-sensitive functions",
		"Define memory budget and track allocations",
	)
	r.Score = 50

	return r
}
