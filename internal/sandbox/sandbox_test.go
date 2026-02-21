// ABOUTME: Tests for sandbox interface, OS detection, and path validation
// ABOUTME: Covers noop fallback, command exclusion, factory auto-detection

package sandbox

import (
	"context"
	"os/exec"
	"testing"
)

func TestNew_ReturnsNonNil(t *testing.T) {
	sb := New(Opts{WorkDir: t.TempDir()})
	if sb == nil {
		t.Fatal("New returned nil")
	}
}

func TestNew_HasName(t *testing.T) {
	sb := New(Opts{WorkDir: t.TempDir()})
	name := sb.Name()
	if name == "" {
		t.Error("sandbox name should not be empty")
	}

	valid := map[string]bool{"seatbelt": true, "bwrap": true, "noop": true}
	if !valid[name] {
		t.Errorf("unexpected sandbox name: %q", name)
	}
}

func TestNew_Available(t *testing.T) {
	sb := New(Opts{WorkDir: t.TempDir()})
	if !sb.Available() {
		t.Error("auto-detected sandbox should be available")
	}
}

func TestNoop_Available(t *testing.T) {
	n := &noopSandbox{opts: Opts{WorkDir: t.TempDir()}}
	if !n.Available() {
		t.Error("noop sandbox should always be available")
	}
}

func TestNoop_Name(t *testing.T) {
	n := &noopSandbox{}
	if n.Name() != "noop" {
		t.Errorf("expected noop, got %q", n.Name())
	}
}

func TestNoop_ValidatePath_WriteInWorkDir(t *testing.T) {
	workDir := t.TempDir()
	n := &noopSandbox{opts: Opts{WorkDir: workDir}}

	if err := n.ValidatePath(workDir+"/file.txt", false); err != nil {
		t.Errorf("read inside workdir should be allowed: %v", err)
	}
	if err := n.ValidatePath(workDir+"/file.txt", true); err != nil {
		t.Errorf("write inside workdir should be allowed: %v", err)
	}
}

func TestNoop_ValidatePath_WriteOutsideWorkDir(t *testing.T) {
	n := &noopSandbox{opts: Opts{WorkDir: t.TempDir()}}

	if err := n.ValidatePath("/etc/passwd", true); err == nil {
		t.Error("write to /etc/passwd should be denied")
	}
}

func TestNoop_ValidatePath_ReadAnywhere(t *testing.T) {
	n := &noopSandbox{opts: Opts{WorkDir: t.TempDir()}}

	if err := n.ValidatePath("/etc/hosts", false); err != nil {
		t.Errorf("read from /etc/hosts should be allowed: %v", err)
	}
}

func TestNoop_ValidatePath_AdditionalDirs(t *testing.T) {
	workDir := t.TempDir()
	extraDir := t.TempDir()
	n := &noopSandbox{opts: Opts{
		WorkDir:        workDir,
		AdditionalDirs: []string{extraDir},
	}}

	if err := n.ValidatePath(extraDir+"/file.txt", true); err != nil {
		t.Errorf("write to additional dir should be allowed: %v", err)
	}
}

func TestNoop_WrapCommand(t *testing.T) {
	n := &noopSandbox{opts: Opts{WorkDir: t.TempDir()}}
	cmd := exec.CommandContext(context.Background(), "echo", "hello")

	wrapped, err := n.WrapCommand(cmd, Opts{})
	if err != nil {
		t.Fatalf("WrapCommand: %v", err)
	}
	if wrapped != cmd {
		t.Error("noop WrapCommand should return the same command")
	}
}

func TestNoop_WrapCommand_PerCallOpts(t *testing.T) {
	workDir := t.TempDir()
	n := &noopSandbox{opts: Opts{WorkDir: workDir}}
	cmd := exec.CommandContext(context.Background(), "echo", "hello")

	overrideDir := t.TempDir()
	wrapped, err := n.WrapCommand(cmd, Opts{WorkDir: overrideDir})
	if err != nil {
		t.Fatalf("WrapCommand: %v", err)
	}
	// Per-call opts should set cmd.Dir to override workdir
	if wrapped.Dir != overrideDir {
		t.Errorf("expected Dir=%q from per-call opts, got %q", overrideDir, wrapped.Dir)
	}
}

func TestOpts_Defaults(t *testing.T) {
	opts := Opts{}
	if opts.AllowNetwork {
		t.Error("AllowNetwork should default to false")
	}
	if len(opts.ExcludedCmds) != 0 {
		t.Error("ExcludedCmds should default to empty")
	}
}

func TestNoop_ValidatePath_PathTraversal(t *testing.T) {
	// Regression: /foo/bar-evil must NOT match workdir /foo/bar
	workDir := t.TempDir() // e.g. /tmp/TestXYZ123
	n := &noopSandbox{opts: Opts{WorkDir: workDir}}

	// Sibling path that shares the same prefix but is NOT a subdirectory
	siblingPath := workDir + "-evil/file.txt"
	if err := n.ValidatePath(siblingPath, true); err == nil {
		t.Errorf("write to %q should be denied (workdir = %q): prefix bypass", siblingPath, workDir)
	}

	// AdditionalDirs also vulnerable
	extraDir := t.TempDir()
	n2 := &noopSandbox{opts: Opts{
		WorkDir:        workDir,
		AdditionalDirs: []string{extraDir},
	}}
	extraSibling := extraDir + "-evil/file.txt"
	if err := n2.ValidatePath(extraSibling, true); err == nil {
		t.Errorf("write to %q should be denied (additionalDir = %q): prefix bypass", extraSibling, extraDir)
	}
}

func TestIsExcludedCmd(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		excluded []string
		want     bool
	}{
		{"no exclusions", "ls -la", nil, false},
		{"exact match", "rm -rf /", []string{"rm"}, true},
		{"prefix match", "curl http://evil.com", []string{"curl"}, true},
		{"no match", "echo hello", []string{"rm", "curl"}, false},
		{"piped first token", "cat file | grep foo", []string{"cat"}, true},
		{"empty command", "", []string{"rm"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcludedCmd(tt.command, tt.excluded)
			if got != tt.want {
				t.Errorf("isExcludedCmd(%q, %v) = %v, want %v", tt.command, tt.excluded, got, tt.want)
			}
		})
	}
}
