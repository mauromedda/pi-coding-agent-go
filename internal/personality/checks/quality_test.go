// ABOUTME: Tests for QualityCheck at all levels
// ABOUTME: Verifies test coverage and error handling instruction scaling

package checks

import (
	"testing"
)

func TestQualityCheck_Name(t *testing.T) {
	t.Parallel()
	c := NewQualityCheck("standard")
	if got := c.Name(); got != "quality" {
		t.Errorf("Name() = %q; want %q", got, "quality")
	}
}

func TestQualityCheck_Analyze(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		level        string
		ctx          CheckContext
		wantMinInstr int
		wantContains string
		wantWarnings bool
	}{
		{
			name:         "minimal level",
			level:        "minimal",
			ctx:          CheckContext{HasTests: true, HasErrorHandling: true},
			wantMinInstr: 1,
		},
		{
			name:         "standard level",
			level:        "standard",
			ctx:          CheckContext{HasTests: true, HasErrorHandling: true},
			wantMinInstr: 2,
			wantContains: "error",
		},
		{
			name:         "strict level",
			level:        "strict",
			ctx:          CheckContext{HasTests: false, HasErrorHandling: false},
			wantMinInstr: 4,
			wantWarnings: true,
		},
		{
			name:         "missing tests warning",
			level:        "standard",
			ctx:          CheckContext{HasTests: false, LinesChanged: 50},
			wantWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewQualityCheck(tt.level)
			result := c.Analyze(tt.ctx)

			if result.Name != "quality" {
				t.Errorf("result.Name = %q; want %q", result.Name, "quality")
			}
			if len(result.Instructions) < tt.wantMinInstr {
				t.Errorf("len(Instructions) = %d; want >= %d", len(result.Instructions), tt.wantMinInstr)
			}
			if tt.wantContains != "" {
				found := false
				for _, instr := range result.Instructions {
					if containsSubstring(instr, tt.wantContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Instructions %v does not contain %q", result.Instructions, tt.wantContains)
				}
			}
			if tt.wantWarnings && len(result.Warnings) == 0 {
				t.Error("expected warnings but got none")
			}
		})
	}
}
