// ABOUTME: Tests for image utilities

package image

import (
	"os"
	"strings"
	"testing"
)

func TestDetectProtocol(t *testing.T) {
	// Clear environment variables for test
	oldIterm := os.Getenv("ITERM_SESSION_ID")
	oldKitty := os.Getenv("KITTY_PID")
	oldTerm := os.Getenv("TERM_PROGRAM")

	// Test macOS default
	os.Unsetenv("ITERM_SESSION_ID")
	os.Unsetenv("KITTY_PID")
	os.Unsetenv("TERM_PROGRAM")
	proto := DetectProtocol()
	if proto != ProtocolITerm2 {
		t.Errorf("Expected default protocol to be ProtocolITerm2 on macOS, got %d", proto)
	}

	// Test iTerm2 detection
	os.Setenv("ITERM_SESSION_ID", "test")
	os.Unsetenv("KITTY_PID")
	os.Unsetenv("TERM_PROGRAM")
	proto = DetectProtocol()
	if proto != ProtocolITerm2 {
		t.Errorf("Expected iTerm2 protocol when ITERM_SESSION_ID set, got %s", proto.String())
	}

	// Test Kitty detection
	os.Unsetenv("ITERM_SESSION_ID")
	os.Setenv("KITTY_PID", "test")
	os.Unsetenv("TERM_PROGRAM")
	proto = DetectProtocol()
	if proto != ProtocolKitty {
		t.Errorf("Expected Kitty protocol when KITTY_PID set, got %s", proto.String())
	}

	// Restore environment
	os.Setenv("ITERM_SESSION_ID", oldIterm)
	os.Setenv("KITTY_PID", oldKitty)
	os.Setenv("TERM_PROGRAM", oldTerm)
}

func TestEncodeImageBase64(t *testing.T) {
	data := []byte("test image data")
	encoded := EncodeImageBase64(data)
	
	expected := "dGVzdCBpbWFnZSBkYXRh"
	if encoded != expected {
		t.Errorf("Expected '%s', got '%s'", expected, encoded)
	}
}

func TestImageRefITerm2(t *testing.T) {
	data := []byte("test")
	
	// Force iTerm2 protocol for test
	os.Setenv("ITERM_SESSION_ID", "test")
	os.Unsetenv("KITTY_PID")
	
	ref := ImageRef(data)
	
	if !strings.HasPrefix(ref, "\x1b]1337;File=base64:") {
		t.Errorf("Expected iTerm2 prefix, got: %s", ref)
	}
	
	// Check forbell character (ASCII 7)
	if !strings.Contains(ref, "\x07") {
		t.Errorf("Expected bell character in iTerm2 ref, got: %s", ref)
	}
}

func TestImageRefKitty(t *testing.T) {
	data := []byte("test")
	
	// Force Kitty protocol for test
	os.Setenv("KITTY_PID", "test")
	os.Unsetenv("ITERM_SESSION_ID")
	
	ref := ImageRef(data)
	
	if !strings.HasPrefix(ref, "\x1b_G") {
		t.Errorf("Expected Kitty prefix, got: %s", ref)
	}
	
	if !strings.HasSuffix(ref, "\x1b\\") {
		t.Errorf("Expected Kitty suffix, got: %s", ref)
	}
}

func TestImagePlaceholder(t *testing.T) {
	data := []byte("test image data")
	placeholder := ImagePlaceholder(data)
	
	expected := "[Image: 15 bytes]"
	if placeholder != expected {
		t.Errorf("Expected '%s', got '%s'", expected, placeholder)
	}
}

func TestEscapeSequence(t *testing.T) {
	data := []byte("test")
	seq := EscapeSequence(data)
	
	if !strings.HasPrefix(seq, "\x1b]1337;File=base64:") {
		t.Errorf("Expected escape sequence prefix, got: %s", seq)
	}
	
	if !strings.HasSuffix(seq, "\x1b\\") {
		t.Errorf("Expected escape sequence suffix, got: %s", seq)
	}
}
