// ABOUTME: macOS-specific tests for seatbelt sandbox profile generation
// ABOUTME: Verifies SBPL profile contains work dirs and read/write rules

//go:build darwin

package sandbox

import (
	"strings"
	"testing"
)

func TestSeatbelt_ProfileGeneration(t *testing.T) {
	s := &seatbeltSandbox{opts: Opts{
		WorkDir:        "/tmp/test-project",
		AdditionalDirs: []string{"/tmp/extra"},
	}}

	profile := s.generateProfile()
	if !strings.Contains(profile, "/tmp/test-project") {
		t.Error("profile should contain work dir")
	}
	if !strings.Contains(profile, "/tmp/extra") {
		t.Error("profile should contain additional dirs")
	}
	if !strings.Contains(profile, "allow file-read*") {
		t.Error("profile should allow reads")
	}
	if !strings.Contains(profile, "allow file-write*") {
		t.Error("profile should allow writes to specific dirs")
	}
}

func TestSeatbelt_ProfileNetwork(t *testing.T) {
	s := &seatbeltSandbox{opts: Opts{
		WorkDir:      "/tmp/test",
		AllowNetwork: true,
	}}

	profile := s.generateProfile()
	if !strings.Contains(profile, "allow network") {
		t.Error("profile should allow network when AllowNetwork is true")
	}

	s2 := &seatbeltSandbox{opts: Opts{
		WorkDir: "/tmp/test",
	}}
	profile2 := s2.generateProfile()
	if strings.Contains(profile2, "allow network") {
		t.Error("profile should not allow network by default")
	}
}

func TestSeatbelt_ProfileCached(t *testing.T) {
	s := &seatbeltSandbox{opts: Opts{WorkDir: "/tmp/test-project"}}

	p1 := s.getProfile()
	p2 := s.getProfile()

	if p1 != p2 {
		t.Error("cached profile should return identical string")
	}
	if p1 == "" {
		t.Error("profile should not be empty")
	}
}

func TestSeatbelt_Available(t *testing.T) {
	s := &seatbeltSandbox{}
	// On macOS, sandbox-exec should be available
	if !s.Available() {
		t.Skip("sandbox-exec not available on this system")
	}
}

func TestSeatbelt_Name(t *testing.T) {
	s := &seatbeltSandbox{}
	if s.Name() != "seatbelt" {
		t.Errorf("expected seatbelt, got %q", s.Name())
	}
}

func TestSeatbelt_ValidatePath(t *testing.T) {
	s := &seatbeltSandbox{opts: Opts{WorkDir: "/tmp/test-project"}}

	if err := s.ValidatePath("/tmp/test-project/file.go", true); err != nil {
		t.Errorf("write in workdir should be allowed: %v", err)
	}
	if err := s.ValidatePath("/etc/passwd", true); err == nil {
		t.Error("write outside workdir should be denied")
	}
	if err := s.ValidatePath("/etc/passwd", false); err != nil {
		t.Errorf("read should always be allowed: %v", err)
	}
}
