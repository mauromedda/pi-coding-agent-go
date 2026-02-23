// ABOUTME: Tests for check factory function and AllCheckNames
// ABOUTME: Verifies NewCheck returns correct types and nil for unknown names

package checks

import (
	"testing"
)

func TestNewCheck_Known(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		checkName string
		level     string
		wantName  string
	}{
		{"security check", "security", "standard", "security"},
		{"performance check", "performance", "standard", "performance"},
		{"quality check", "quality", "standard", "quality"},
		{"architecture check", "architecture", "standard", "architecture"},
		{"factual check", "factual", "standard", "factual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewCheck(tt.checkName, tt.level)
			if c == nil {
				t.Fatalf("NewCheck(%q, %q) = nil; want non-nil", tt.checkName, tt.level)
			}
			if got := c.Name(); got != tt.wantName {
				t.Errorf("Name() = %q; want %q", got, tt.wantName)
			}
		})
	}
}

func TestNewCheck_Unknown(t *testing.T) {
	t.Parallel()
	c := NewCheck("nonexistent", "standard")
	if c != nil {
		t.Errorf("NewCheck(\"nonexistent\", \"standard\") = %v; want nil", c)
	}
}

func TestAllCheckNames(t *testing.T) {
	t.Parallel()
	names := AllCheckNames()
	if len(names) != 5 {
		t.Errorf("len(AllCheckNames()) = %d; want 5", len(names))
	}

	expected := map[string]bool{
		"security":     true,
		"performance":  true,
		"quality":      true,
		"architecture": true,
		"factual":      true,
	}
	for _, n := range names {
		if !expected[n] {
			t.Errorf("unexpected check name: %q", n)
		}
	}
}
