// ABOUTME: Unix-specific process group management for hook commands
// ABOUTME: Sets Setpgid and kills process groups with SIGKILL on timeout

//go:build unix

package hooks

import (
	"os/exec"
	"syscall"
)

// setProcGroup configures the command to run in its own process group.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcGroup kills the entire process group of the command.
func killProcGroup(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return nil
}
