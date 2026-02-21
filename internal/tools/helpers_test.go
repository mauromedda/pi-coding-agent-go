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
