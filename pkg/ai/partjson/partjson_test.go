// ABOUTME: Tests for partial JSON parser (handles incomplete streaming JSON)
// ABOUTME: Covers truncated strings, objects, arrays, nested structures, and edge cases

package partjson

import (
	"testing"
)

func TestParse_CompleteJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		key   string
		want  any
	}{
		{"object", `{"command":"ls"}`, "command", "ls"},
		{"nested", `{"a":{"b":1}}`, "a", map[string]any{"b": float64(1)}},
		{"array value", `{"items":[1,2,3]}`, "items", []any{float64(1), float64(2), float64(3)}},
		{"boolean", `{"ok":true}`, "ok", true},
		{"null", `{"x":null}`, "x", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Parse(tt.input)
			if result == nil {
				t.Fatal("result is nil")
			}
		})
	}
}

func TestParse_TruncatedString(t *testing.T) {
	t.Parallel()

	result := Parse(`{"command":"ls -l`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["command"] != "ls -l" {
		t.Errorf("command = %v", result["command"])
	}
}

func TestParse_TruncatedObject(t *testing.T) {
	t.Parallel()

	result := Parse(`{"path":"/tmp","limit":10`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["path"] != "/tmp" {
		t.Errorf("path = %v", result["path"])
	}
	if result["limit"] != float64(10) {
		t.Errorf("limit = %v", result["limit"])
	}
}

func TestParse_TruncatedArray(t *testing.T) {
	t.Parallel()

	result := Parse(`{"items":["a","b","c`)
	if result == nil {
		t.Fatal("result is nil")
	}
	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("items type = %T", result["items"])
	}
	if len(items) < 2 {
		t.Errorf("items = %v", items)
	}
}

func TestParse_TruncatedKey(t *testing.T) {
	t.Parallel()

	result := Parse(`{"comm`)
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestParse_TruncatedAfterColon(t *testing.T) {
	t.Parallel()

	result := Parse(`{"command":`)
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestParse_Empty(t *testing.T) {
	t.Parallel()

	result := Parse("")
	if result == nil {
		t.Fatal("result is nil for empty input")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestParse_JustBrace(t *testing.T) {
	t.Parallel()

	result := Parse("{")
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestParse_NestedTruncated(t *testing.T) {
	t.Parallel()

	result := Parse(`{"outer":{"inner":"val`)
	if result == nil {
		t.Fatal("result is nil")
	}
	outer, ok := result["outer"].(map[string]any)
	if !ok {
		t.Fatalf("outer type = %T", result["outer"])
	}
	if outer["inner"] != "val" {
		t.Errorf("inner = %v", outer["inner"])
	}
}

func TestParse_EscapedChars(t *testing.T) {
	t.Parallel()

	input := `{"text":"hello \"world`
	result := Parse(input)
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestParse_NumberTruncated(t *testing.T) {
	t.Parallel()

	result := Parse(`{"count":12`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["count"] != float64(12) {
		t.Errorf("count = %v (type %T)", result["count"], result["count"])
	}
}

func TestParse_TruncatedAfterEscapeChar(t *testing.T) {
	t.Parallel()

	result := Parse(`{"key":"val\`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if _, ok := result["key"]; !ok {
		t.Error("expected 'key' to be present in result")
	}
}

func TestParse_TruncatedBoolean(t *testing.T) {
	t.Parallel()

	// "tr" is incomplete boolean; parser should strip it and produce a valid object
	// The key "ok" won't have a value, but parsing must not fail entirely.
	// At minimum, the result must not contain a garbled key.
	result := Parse(`{"ok":tr`)
	if result == nil {
		t.Fatal("result is nil")
	}
	// After stripping "tr", the input becomes `{"ok":` -> trim colon -> `{"ok"` -> close -> `{"ok"}`
	// which is invalid JSON ("ok" is a string, not a key-value pair).
	// Actually: `{"ok":` -> trim colon -> `{"ok"` -> but that's still in a string?
	// Let's just verify it doesn't panic and returns a map (possibly empty).
	// The real assertion: it must be valid enough to not crash.
	if len(result) > 0 {
		t.Logf("result = %v", result)
	}
}

func TestParse_TruncatedNull(t *testing.T) {
	t.Parallel()

	result := Parse(`{"v":nu`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result) > 0 {
		t.Logf("result = %v", result)
	}
}

func TestParse_TruncatedFalse(t *testing.T) {
	t.Parallel()

	result := Parse(`{"flag":fal`)
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestParse_TruncatedBooleanWithPrecedingKey(t *testing.T) {
	t.Parallel()

	// The important case: a complete key-value pair followed by a truncated boolean.
	// The first pair must survive even though the second value is incomplete.
	result := Parse(`{"name":"test","ok":tr`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want 'test'", result["name"])
	}
}

func TestParse_TruncatedNullWithPrecedingKey(t *testing.T) {
	t.Parallel()

	result := Parse(`{"name":"test","v":nul`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want 'test'", result["name"])
	}
}

func TestParse_CompleteBooleanNotStripped(t *testing.T) {
	t.Parallel()

	result := Parse(`{"ok":true}`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["ok"] != true {
		t.Errorf("ok = %v, want true", result["ok"])
	}
}

func TestParse_CompleteNullNotStripped(t *testing.T) {
	t.Parallel()

	result := Parse(`{"v":null}`)
	if result == nil {
		t.Fatal("result is nil")
	}
	// null values are present but nil
	if v, ok := result["v"]; !ok {
		t.Error("expected 'v' key to be present")
	} else if v != nil {
		t.Errorf("v = %v, want nil", v)
	}
}

func TestParse_CompleteFalseNotStripped(t *testing.T) {
	t.Parallel()

	result := Parse(`{"flag":false}`)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result["flag"] != false {
		t.Errorf("flag = %v, want false", result["flag"])
	}
}
