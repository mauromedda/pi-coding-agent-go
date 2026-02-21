// ABOUTME: Tests for ANSI stripping, extraction, and SGR tracking
// ABOUTME: Covers CSI, OSC, and simple ESC sequences

package width

import (
	"reflect"
	"testing"
)

func TestStripANSI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no ansi", input: "plain text", want: "plain text"},
		{name: "sgr color", input: "\x1b[31mred\x1b[0m", want: "red"},
		{name: "bold", input: "\x1b[1mbold\x1b[0m", want: "bold"},
		{name: "multiple sgr", input: "\x1b[31;1;4mstuff\x1b[0m", want: "stuff"},
		{name: "osc", input: "\x1b]0;title\x07text", want: "text"},
		{name: "cursor", input: "\x1b[10;20Hhere", want: "here"},
		{name: "empty", input: "", want: ""},
		{name: "only escape", input: "\x1b[0m", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractANSI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "no ansi", input: "plain", want: nil},
		{name: "one csi", input: "\x1b[31mred\x1b[0m", want: []string{"\x1b[31m", "\x1b[0m"}},
		{name: "osc + bel", input: "\x1b]0;title\x07", want: []string{"\x1b]0;title\x07"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractANSI(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractANSI(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestActiveSGR(t *testing.T) {
	t.Parallel()

	var sgr ActiveSGR
	sgr.Apply("\x1b[31m")
	sgr.Apply("\x1b[1m")

	got := sgr.String()
	if got != "\x1b[31m\x1b[1m" {
		t.Errorf("SGR.String() = %q, want %q", got, "\x1b[31m\x1b[1m")
	}

	sgr.Apply("\x1b[0m")
	if s := sgr.String(); s != "" {
		t.Errorf("after reset, SGR.String() = %q, want empty", s)
	}
}
