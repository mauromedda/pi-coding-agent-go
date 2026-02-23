// ABOUTME: Tests for PerformanceCheck at all levels
// ABOUTME: Verifies instruction scaling and performance-critical context handling

package checks

import (
	"testing"
)

func TestPerformanceCheck_Name(t *testing.T) {
	t.Parallel()
	c := NewPerformanceCheck("standard")
	if got := c.Name(); got != "performance" {
		t.Errorf("Name() = %q; want %q", got, "performance")
	}
}

func TestPerformanceCheck_Analyze(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		level        string
		ctx          CheckContext
		wantMinInstr int
		wantContains string
	}{
		{
			name:         "minimal level",
			level:        "minimal",
			ctx:          CheckContext{},
			wantMinInstr: 1,
			wantContains: "N+1",
		},
		{
			name:         "standard level",
			level:        "standard",
			ctx:          CheckContext{IsPerformanceCritical: true},
			wantMinInstr: 3,
			wantContains: "pre-allocat",
		},
		{
			name:         "strict level",
			level:        "strict",
			ctx:          CheckContext{IsPerformanceCritical: true, LinesChanged: 200},
			wantMinInstr: 5,
			wantContains: "profil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewPerformanceCheck(tt.level)
			result := c.Analyze(tt.ctx)

			if result.Name != "performance" {
				t.Errorf("result.Name = %q; want %q", result.Name, "performance")
			}
			if result.Level != tt.level {
				t.Errorf("result.Level = %q; want %q", result.Level, tt.level)
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
		})
	}
}
