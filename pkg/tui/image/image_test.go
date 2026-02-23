// ABOUTME: Tests for clipboard utilities, base64 encoding, and placeholders
// ABOUTME: Protocol-specific tests moved to protocol_test.go, kitty_test.go, iterm2_test.go

package image

import (
	"testing"
)

func TestEncodeImageBase64(t *testing.T) {
	data := []byte("test image data")
	encoded := EncodeImageBase64(data)

	expected := "dGVzdCBpbWFnZSBkYXRh"
	if encoded != expected {
		t.Errorf("got %q, want %q", encoded, expected)
	}
}

func TestImagePlaceholder(t *testing.T) {
	data := []byte("test image data")
	placeholder := ImagePlaceholder(data)

	expected := "[Image: 15 bytes]"
	if placeholder != expected {
		t.Errorf("got %q, want %q", placeholder, expected)
	}
}
