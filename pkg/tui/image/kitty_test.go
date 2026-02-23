// ABOUTME: Tests for the Kitty graphics protocol encoder
// ABOUTME: Verifies chunked base64 output, escape sequences, and edge cases

package image

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncodeKitty_SmallPayload(t *testing.T) {
	// Small PNG that fits in a single chunk (< 4096 base64 chars)
	data := make([]byte, 100)
	result := EncodeKitty(data, 40, 10)

	// Should start with the Kitty APC escape
	if !strings.HasPrefix(result, "\x1b_G") {
		t.Error("expected Kitty APC prefix \\x1b_G")
	}
	// Single chunk: m=0 (no more data)
	if !strings.Contains(result, "m=0") {
		t.Error("expected m=0 for single chunk")
	}
	// Should contain the control params
	if !strings.Contains(result, "a=T") {
		t.Error("expected a=T (transmit action)")
	}
	if !strings.Contains(result, "f=100") {
		t.Error("expected f=100 (PNG format)")
	}
	if !strings.Contains(result, "q=2") {
		t.Error("expected q=2 (suppress response)")
	}
	// Should end with ST
	if !strings.HasSuffix(result, "\x1b\\") {
		t.Error("expected Kitty ST suffix \\x1b\\\\")
	}
}

func TestEncodeKitty_LargePayload(t *testing.T) {
	// Create data that produces more than 4096 base64 chars
	// 4096 base64 chars = 3072 raw bytes, so use 4000 bytes
	data := make([]byte, 4000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	result := EncodeKitty(data, 80, 24)

	chunks := strings.Split(result, "\x1b\\")
	// Last element is empty after the final \x1b\\
	chunks = chunks[:len(chunks)-1]

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// First chunk: has full header with m=1
	if !strings.Contains(chunks[0], "a=T") {
		t.Error("first chunk should contain a=T")
	}
	if !strings.Contains(chunks[0], "m=1") {
		t.Error("first chunk should have m=1 (more data follows)")
	}

	// Last chunk: m=0
	last := chunks[len(chunks)-1]
	if !strings.Contains(last, "m=0") {
		t.Errorf("last chunk should have m=0, got: %s", last)
	}
}

func TestEncodeKitty_Base64Content(t *testing.T) {
	data := []byte("PNG image data here")
	result := EncodeKitty(data, 40, 10)

	expected := base64.StdEncoding.EncodeToString(data)
	// The base64 payload should appear after the semicolon separator
	if !strings.Contains(result, ";"+expected) {
		t.Error("expected base64 payload in output")
	}
}

func TestEncodeKitty_EmptyData(t *testing.T) {
	result := EncodeKitty(nil, 40, 10)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}

	result = EncodeKitty([]byte{}, 40, 10)
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}
