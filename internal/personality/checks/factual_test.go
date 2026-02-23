// ABOUTME: Tests for FactualCheck at all levels
// ABOUTME: Verifies documentation verification instructions scale with level

package checks

import (
	"testing"
)

func TestFactualCheck_Name(t *testing.T) {
	t.Parallel()
	c := NewFactualCheck("standard")
	if got := c.Name(); got != "factual" {
		t.Errorf("Name() = %q; want %q", got, "factual")
	}
}

func TestFactualCheck_Analyze(t *testing.T) {
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
			wantContains: "verif",
		},
		{
			name:         "standard level",
			level:        "standard",
			ctx:          CheckContext{Languages: []string{"go"}},
			wantMinInstr: 2,
			wantContains: "API",
		},
		{
			name:         "strict level",
			level:        "strict",
			ctx:          CheckContext{Languages: []string{"go", "python"}},
			wantMinInstr: 4,
			wantContains: "deprecat",
		},
		{
			name:         "paranoid level",
			level:        "paranoid",
			ctx:          CheckContext{},
			wantMinInstr: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewFactualCheck(tt.level)
			result := c.Analyze(tt.ctx)

			if result.Name != "factual" {
				t.Errorf("result.Name = %q; want %q", result.Name, "factual")
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
