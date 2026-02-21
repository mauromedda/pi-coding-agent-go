// ABOUTME: macOS seatbelt sandbox using sandbox-exec with custom SBPL profiles
// ABOUTME: Allows reads everywhere, restricts writes to WorkDir + AdditionalDirs

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// seatbeltSandbox uses macOS sandbox-exec with a custom profile.
type seatbeltSandbox struct {
	opts Opts
}

func (s *seatbeltSandbox) WrapCommand(cmd *exec.Cmd, _ Opts) (*exec.Cmd, error) {
	profile := s.generateProfile()

	args := []string{"-p", profile, cmd.Path}
	args = append(args, cmd.Args[1:]...)

	wrapped := exec.CommandContext(context.Background(), "sandbox-exec", args...)
	wrapped.Dir = cmd.Dir
	wrapped.Env = cmd.Env
	wrapped.Stdin = cmd.Stdin
	wrapped.Stdout = cmd.Stdout
	wrapped.Stderr = cmd.Stderr

	return wrapped, nil
}

func (s *seatbeltSandbox) ValidatePath(path string, write bool) error {
	if !write {
		return nil
	}
	return validateWritePath(path, s.opts)
}

func (s *seatbeltSandbox) Available() bool {
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

func (s *seatbeltSandbox) Name() string { return "seatbelt" }

// generateProfile creates a Scheme-based Apple Sandbox Profile Language (SBPL) profile.
func (s *seatbeltSandbox) generateProfile() string {
	var b strings.Builder

	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")
	b.WriteString("(allow process*)\n")
	b.WriteString("(allow signal)\n")
	b.WriteString("(allow sysctl-read)\n")
	b.WriteString("(allow mach-lookup)\n")
	b.WriteString("(allow file-read*)\n")

	writeDirs := []string{s.opts.WorkDir}
	writeDirs = append(writeDirs, s.opts.AdditionalDirs...)
	writeDirs = append(writeDirs, "/tmp", "/private/tmp", "/var/folders")

	for _, dir := range writeDirs {
		abs, _ := filepath.Abs(dir)
		if abs == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("(allow file-write* (subpath %q))\n", abs))
	}

	if s.opts.AllowNetwork {
		b.WriteString("(allow network*)\n")
	}

	return b.String()
}
