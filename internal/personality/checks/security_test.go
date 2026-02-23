// ABOUTME: Tests for SecurityCheck at all four levels
// ABOUTME: Verifies instruction count and content scaling with rigor level

package checks

import (
	"strings"
	"testing"
)

func TestSecurityCheck_Name(t *testing.T) {
	t.Parallel()
	c := NewSecurityCheck("standard")
	if got := c.Name(); got != "security" {
		t.Errorf("Name() = %q; want %q", got, "security")
	}
}

func TestSecurityCheck_Analyze(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		level            string
		ctx              CheckContext
		wantMinInstr     int
		wantContains     string
		wantMinScore     int
		wantMaxScore     int
		wantWarningCount int
	}{
		{
			name:         "minimal level basic",
			level:        "minimal",
			ctx:          CheckContext{IsSecurityRelated: false},
			wantMinInstr: 1,
			wantContains: "input validation",
			wantMinScore: 70,
			wantMaxScore: 100,
		},
		{
			name:         "standard level",
			level:        "standard",
			ctx:          CheckContext{IsSecurityRelated: true},
			wantMinInstr: 3,
			wantContains: "OWASP",
			wantMinScore: 50,
			wantMaxScore: 100,
		},
		{
			name:         "strict level",
			level:        "strict",
			ctx:          CheckContext{IsSecurityRelated: true, FilesChanged: 5},
			wantMinInstr: 5,
			wantContains: "secrets",
			wantMinScore: 40,
			wantMaxScore: 100,
		},
		{
			name:         "paranoid level",
			level:        "paranoid",
			ctx:          CheckContext{IsSecurityRelated: true, FilesChanged: 10},
			wantMinInstr: 7,
			wantContains: "threat model",
			wantMinScore: 30,
			wantMaxScore: 100,
		},
		{
			name:             "security related context adds warnings",
			level:            "standard",
			ctx:              CheckContext{IsSecurityRelated: true, HasErrorHandling: false},
			wantMinInstr:     3,
			wantContains:     "OWASP",
			wantWarningCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewSecurityCheck(tt.level)
			result := c.Analyze(tt.ctx)

			if result.Name != "security" {
				t.Errorf("result.Name = %q; want %q", result.Name, "security")
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
			if tt.wantMinScore > 0 && result.Score < tt.wantMinScore {
				t.Errorf("Score = %d; want >= %d", result.Score, tt.wantMinScore)
			}
			if tt.wantMaxScore > 0 && result.Score > tt.wantMaxScore {
				t.Errorf("Score = %d; want <= %d", result.Score, tt.wantMaxScore)
			}
			if tt.wantWarningCount > 0 && len(result.Warnings) < tt.wantWarningCount {
				t.Errorf("len(Warnings) = %d; want >= %d", len(result.Warnings), tt.wantWarningCount)
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
