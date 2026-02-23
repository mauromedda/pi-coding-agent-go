// ABOUTME: Security check with 4 levels: minimal, standard, strict, paranoid
// ABOUTME: Generates prompt instructions for input validation, auth, injection prevention

package checks

// SecurityCheck analyzes security concerns at configurable rigor levels.
type SecurityCheck struct {
	level string
}

// NewSecurityCheck creates a SecurityCheck at the given level.
func NewSecurityCheck(level string) *SecurityCheck {
	return &SecurityCheck{level: level}
}

// Name returns the check name.
func (c *SecurityCheck) Name() string { return "security" }

// Analyze runs security analysis and returns instructions based on level and context.
func (c *SecurityCheck) Analyze(ctx CheckContext) CheckResult {
	r := CheckResult{
		Name:  "security",
		Level: c.level,
		Score: 100,
	}

	// Minimal: basic input validation
	r.Instructions = append(r.Instructions, "Apply input validation to all user-provided data")
	r.Score = 80

	if c.level == "minimal" {
		return r
	}

	// Standard: OWASP top 10, auth, sanitization
	r.Instructions = append(r.Instructions,
		"Check against OWASP Top 10 vulnerabilities",
		"Verify authentication and authorization on all endpoints",
		"Sanitize user input to prevent injection attacks",
	)
	r.Score = 70

	if ctx.IsSecurityRelated && !ctx.HasErrorHandling {
		r.Warnings = append(r.Warnings, "Security-related code lacks error handling")
	}

	if c.level == "standard" {
		return r
	}

	// Strict: + secrets scanning, dependency audit, CSP
	r.Instructions = append(r.Instructions,
		"Scan for hardcoded secrets and credentials",
		"Audit dependencies for known vulnerabilities",
		"Set Content-Security-Policy headers where applicable",
	)
	r.Score = 50

	if c.level == "strict" {
		return r
	}

	// Paranoid: + threat modeling, zero-trust, formal verification
	r.Instructions = append(r.Instructions,
		"Perform threat modeling for all new attack surfaces",
		"Apply zero-trust assertions: verify every request explicitly",
		"Consider formal verification for critical security paths",
	)
	r.Score = 40

	return r
}
