// ABOUTME: Tests for shared helper functions: parameter extraction and bounds checking
// ABOUTME: Covers type-safe accessors, overflow guards, and edge cases

package tools

import (
	"fmt"
	"math"
	"testing"
)

func TestIntParam_Normal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     map[string]any
		key        string
		defaultVal int
		want       int
	}{
		{"float64 value", map[string]any{"n": float64(42)}, "n", 0, 42},
		{"int value", map[string]any{"n": 7}, "n", 0, 7},
		{"missing key", map[string]any{}, "n", 99, 99},
		{"wrong type", map[string]any{"n": "hello"}, "n", 99, 99},
		{"negative float64", map[string]any{"n": float64(-5)}, "n", 0, -5},
		{"zero", map[string]any{"n": float64(0)}, "n", 99, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := intParam(tt.params, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("intParam() = %d; want %d", got, tt.want)
			}
		})
	}
}

func TestIntParam_Overflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		value      float64
		defaultVal int
		want       int
	}{
		{"NaN returns default", math.NaN(), 42, 42},
		{"positive Inf returns default", math.Inf(1), 42, 42},
		{"negative Inf returns default", math.Inf(-1), 42, 42},
		{"exceeds MaxInt returns default", float64(math.MaxInt64) * 2, 42, 42},
		{"below MinInt returns default", float64(math.MinInt64) * 2, 42, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := map[string]any{"n": tt.value}
			got := intParam(params, "n", tt.defaultVal)
			if got != tt.want {
				t.Errorf("intParam() = %d; want %d", got, tt.want)
			}
		})
	}
}

func TestRequireStringParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{"present", map[string]any{"k": "val"}, "k", "val", false},
		{"missing", map[string]any{}, "k", "", true},
		{"wrong type", map[string]any{"k": 42}, "k", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := requireStringParam(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q; want %q", got, tt.want)
			}
		})
	}
}

func TestBoolParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     map[string]any
		key        string
		defaultVal bool
		want       bool
	}{
		{"true", map[string]any{"b": true}, "b", false, true},
		{"false", map[string]any{"b": false}, "b", true, false},
		{"missing", map[string]any{}, "b", true, true},
		{"wrong type", map[string]any{"b": "yes"}, "b", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := boolParam(tt.params, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("boolParam() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestStringParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     map[string]any
		key        string
		defaultVal string
		want       string
	}{
		{"present", map[string]any{"s": "val"}, "s", "def", "val"},
		{"missing", map[string]any{}, "s", "def", "def"},
		{"wrong type", map[string]any{"s": 42}, "s", "def", "def"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stringParam(tt.params, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("stringParam() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestErrResult(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("test error")
	r := errResult(err)
	if !r.IsError {
		t.Error("expected IsError = true")
	}
	if r.Content != "test error" {
		t.Errorf("got %q; want %q", r.Content, "test error")
	}
}

func TestRequireStringSliceParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  map[string]any
		key     string
		want    []string
		wantErr bool
	}{
		{
			"valid slice",
			map[string]any{"paths": []any{"a.go", "b.go"}},
			"paths", []string{"a.go", "b.go"}, false,
		},
		{
			"missing key",
			map[string]any{},
			"paths", nil, true,
		},
		{
			"wrong type (string)",
			map[string]any{"paths": "not-a-slice"},
			"paths", nil, true,
		},
		{
			"empty slice",
			map[string]any{"paths": []any{}},
			"paths", nil, true,
		},
		{
			"non-string element",
			map[string]any{"paths": []any{"ok", 42}},
			"paths", nil, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := requireStringSliceParam(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Fatalf("len = %d; want %d", len(got), len(tt.want))
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("[%d] = %q; want %q", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestShouldSkipDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{"git", ".git", true},
		{"vendor", "vendor", true},
		{"node_modules", "node_modules", true},
		{"pycache", "__pycache__", true},
		{"regular dir", "internal", false},
		{"src", "src", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldSkipDir(tt.dir); got != tt.want {
				t.Errorf("shouldSkipDir(%q) = %v; want %v", tt.dir, got, tt.want)
			}
		})
	}
}
