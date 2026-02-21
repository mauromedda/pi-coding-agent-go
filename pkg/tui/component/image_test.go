// ABOUTME: Tests for the Image component covering Kitty and iTerm2 protocols.
// ABOUTME: Validates base64 encoding, chunked transmission, protocol selection, and rendering.

package component

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func TestNewImage(t *testing.T) {
	t.Parallel()

	data := []byte("fake-png-data")
	img := NewImage(data)

	if img == nil {
		t.Fatal("NewImage returned nil")
	}
}

func TestImage_Render_Kitty(t *testing.T) {
	t.Parallel()

	data := []byte("test-image-data")
	img := NewImage(data)
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)

	if buf.Len() == 0 {
		t.Fatal("expected at least one line of output")
	}

	// Kitty protocol starts with ESC_APC G (which is \x1b_G)
	output := strings.Join(buf.Lines, "")
	if !strings.Contains(output, "\x1b_G") {
		t.Errorf("expected Kitty escape sequence (\\x1b_G), got: %q", output)
	}

	// Must end with ST (\x1b\\)
	if !strings.Contains(output, "\x1b\\") {
		t.Errorf("expected string terminator (\\x1b\\\\), got: %q", output)
	}

	// Must contain base64-encoded data
	encoded := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(output, encoded) {
		t.Error("output does not contain base64-encoded image data")
	}
}

func TestImage_Render_ITerm2(t *testing.T) {
	t.Parallel()

	data := []byte("test-image-data")
	img := NewImage(data)
	img.SetProtocol(ProtocolITerm2)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)

	if buf.Len() == 0 {
		t.Fatal("expected at least one line of output")
	}

	output := strings.Join(buf.Lines, "")

	// iTerm2 protocol: ESC ] 1337 ; File=
	if !strings.Contains(output, "\x1b]1337;File=") {
		t.Errorf("expected iTerm2 escape sequence, got: %q", output)
	}

	// Must contain size parameter
	if !strings.Contains(output, "size=") {
		t.Errorf("expected size parameter in iTerm2 output")
	}

	// Must contain inline=1
	if !strings.Contains(output, "inline=1") {
		t.Errorf("expected inline=1 in iTerm2 output")
	}

	// Must end with BEL (\a) or ST (\x1b\\)
	if !strings.Contains(output, "\a") && !strings.Contains(output, "\x1b\\") {
		t.Errorf("expected BEL or ST terminator")
	}
}

func TestImage_Render_Kitty_Chunked(t *testing.T) {
	t.Parallel()

	// Create data large enough to require chunking (>4096 bytes of base64 output)
	largeData := make([]byte, 4096)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	img := NewImage(largeData)
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)

	output := strings.Join(buf.Lines, "")

	// For chunked data, there should be multiple APC sequences
	// The first chunk has m=1 (more data follows), the last has m=0
	if !strings.Contains(output, "m=1") {
		t.Error("expected m=1 in chunked output indicating more data")
	}
	if !strings.Contains(output, "m=0") {
		t.Error("expected m=0 in final chunk indicating end of data")
	}
}

func TestImage_Render_Kitty_SmallData(t *testing.T) {
	t.Parallel()

	// Small data fits in a single chunk
	smallData := []byte("tiny")
	img := NewImage(smallData)
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)

	output := strings.Join(buf.Lines, "")

	// Single chunk: no m=1, only m=0
	if strings.Contains(output, "m=1") {
		t.Error("small data should not have continuation chunks")
	}
	if !strings.Contains(output, "m=0") {
		t.Error("expected m=0 for single chunk")
	}
}

func TestImage_SetProtocol(t *testing.T) {
	t.Parallel()

	img := NewImage([]byte("data"))

	img.SetProtocol(ProtocolKitty)
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	img.Render(buf, 80)
	kittyOutput := strings.Join(buf.Lines, "")

	img.SetProtocol(ProtocolITerm2)
	buf.Reset()
	img.Render(buf, 80)
	itermOutput := strings.Join(buf.Lines, "")

	if kittyOutput == itermOutput {
		t.Error("expected different output for different protocols")
	}
}

func TestImage_Invalidate(t *testing.T) {
	t.Parallel()

	img := NewImage([]byte("data"))
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)
	firstLen := buf.Len()

	buf.Reset()
	img.Invalidate()
	img.Render(buf, 80)
	secondLen := buf.Len()

	if firstLen != secondLen {
		t.Errorf("expected same output after invalidate, got %d vs %d lines", firstLen, secondLen)
	}
}

func TestImage_SetData(t *testing.T) {
	t.Parallel()

	img := NewImage([]byte("first"))
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)
	firstOutput := strings.Join(buf.Lines, "")

	img.SetData([]byte("second"))
	buf.Reset()
	img.Render(buf, 80)
	secondOutput := strings.Join(buf.Lines, "")

	if firstOutput == secondOutput {
		t.Error("expected different output after SetData")
	}

	encodedSecond := base64.StdEncoding.EncodeToString([]byte("second"))
	if !strings.Contains(secondOutput, encodedSecond) {
		t.Error("output should contain base64 of new data")
	}
}

func TestDetectProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		termProg string
		want     ImageProtocol
	}{
		{name: "kitty terminal", termProg: "kitty", want: ProtocolKitty},
		{name: "iTerm2", termProg: "iTerm.app", want: ProtocolITerm2},
		{name: "WezTerm", termProg: "WezTerm", want: ProtocolKitty},
		{name: "unknown terminal", termProg: "xterm", want: ProtocolKitty},
		{name: "empty", termProg: "", want: ProtocolKitty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DetectProtocol(tt.termProg)
			if got != tt.want {
				t.Errorf("DetectProtocol(%q) = %v, want %v", tt.termProg, got, tt.want)
			}
		})
	}
}

func TestImage_Render_EmptyData(t *testing.T) {
	t.Parallel()

	img := NewImage([]byte{})
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)

	// Empty data should produce no output
	if buf.Len() != 0 {
		t.Errorf("expected 0 lines for empty data, got %d", buf.Len())
	}
}

func TestImage_Render_NilData(t *testing.T) {
	t.Parallel()

	img := NewImage(nil)
	img.SetProtocol(ProtocolKitty)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	img.Render(buf, 80)

	if buf.Len() != 0 {
		t.Errorf("expected 0 lines for nil data, got %d", buf.Len())
	}
}

func TestImageProtocol_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    ImageProtocol
		want string
	}{
		{name: "kitty", p: ProtocolKitty, want: "kitty"},
		{name: "iterm2", p: ProtocolITerm2, want: "iterm2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.p.String()
			if got != tt.want {
				t.Errorf("ImageProtocol.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
