// ABOUTME: Tests for the iTerm2 inline images protocol encoder
// ABOUTME: Verifies escape sequence structure, base64 payload, and parameters

package image

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestEncodeITerm2_Basic(t *testing.T) {
	data := []byte("fake png data")
	result := EncodeITerm2(data, "auto")

	// OSC 1337 prefix
	if !strings.HasPrefix(result, "\x1b]1337;File=") {
		t.Error("expected OSC 1337 prefix")
	}
	// BEL terminator
	if !strings.HasSuffix(result, "\a") {
		t.Error("expected BEL (\\a) terminator")
	}
	// inline=1
	if !strings.Contains(result, "inline=1") {
		t.Error("expected inline=1 parameter")
	}
	// size parameter
	if !strings.Contains(result, fmt.Sprintf("size=%d", len(data))) {
		t.Error("expected size parameter")
	}
	// width parameter
	if !strings.Contains(result, "width=auto") {
		t.Error("expected width=auto parameter")
	}
}

func TestEncodeITerm2_Base64Content(t *testing.T) {
	data := []byte("test image payload")
	result := EncodeITerm2(data, "80")

	expected := base64.StdEncoding.EncodeToString(data)
	// Base64 comes after the colon separator
	if !strings.Contains(result, ":"+expected+"\a") {
		t.Error("expected base64 payload before BEL")
	}
}

func TestEncodeITerm2_EmptyData(t *testing.T) {
	result := EncodeITerm2(nil, "auto")
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}

	result = EncodeITerm2([]byte{}, "auto")
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}

func TestEncodeITerm2_WidthParam(t *testing.T) {
	data := []byte("x")
	tests := []struct {
		width string
		want  string
	}{
		{"auto", "width=auto"},
		{"80", "width=80"},
		{"50%", "width=50%"},
	}
	for _, tt := range tests {
		t.Run(tt.width, func(t *testing.T) {
			result := EncodeITerm2(data, tt.width)
			if !strings.Contains(result, tt.want) {
				t.Errorf("expected %q in output, got %q", tt.want, result)
			}
		})
	}
}
