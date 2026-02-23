// ABOUTME: Tests for ArchitectureCheck at all levels
// ABOUTME: Verifies structural constraint instructions scale with level

package checks

import (
	"testing"
)

func TestArchitectureCheck_Name(t *testing.T) {
	t.Parallel()
	c := NewArchitectureCheck("standard")
	if got := c.Name(); got != "architecture" {
		t.Errorf("Name() = %q; want %q", got, "architecture")
	}
}

func TestArchitectureCheck_Analyze(t *testing.T) {
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
			ctx:          CheckContext{FilesChanged: 1},
			wantMinInstr: 1,
		},
		{
			name:         "standard level",
			level:        "standard",
			ctx:          CheckContext{FilesChanged: 5},
			wantMinInstr: 2,
			wantContains: "dependenc",
		},
		{
			name:         "strict level",
			level:        "strict",
			ctx:          CheckContext{FilesChanged: 10, Languages: []string{"go", "python"}},
			wantMinInstr: 4,
			wantContains: "interface",
		},
		{
			name:         "paranoid level",
			level:        "paranoid",
			ctx:          CheckContext{FilesChanged: 20},
			wantMinInstr: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewArchitectureCheck(tt.level)
			result := c.Analyze(tt.ctx)

			if result.Name != "architecture" {
				t.Errorf("result.Name = %q; want %q", result.Name, "architecture")
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
