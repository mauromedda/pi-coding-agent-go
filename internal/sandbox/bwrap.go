// ABOUTME: Linux bubblewrap sandbox using bwrap for filesystem isolation
// ABOUTME: Read-only bind of /, writable bind for WorkDir + AdditionalDirs

package sandbox

import (
	"context"
	"os/exec"
	"path/filepath"
)

// bwrapSandbox uses Linux bubblewrap for isolation.
type bwrapSandbox struct {
	opts Opts
}

func (b *bwrapSandbox) WrapCommand(cmd *exec.Cmd, _ Opts) (*exec.Cmd, error) {
	bwrapArgs := b.buildArgs()
	bwrapArgs = append(bwrapArgs, cmd.Path)
	bwrapArgs = append(bwrapArgs, cmd.Args[1:]...)

	wrapped := exec.CommandContext(context.Background(), "bwrap", bwrapArgs...)
	wrapped.Dir = cmd.Dir
	wrapped.Env = cmd.Env
	wrapped.Stdin = cmd.Stdin
	wrapped.Stdout = cmd.Stdout
	wrapped.Stderr = cmd.Stderr

	return wrapped, nil
}

func (b *bwrapSandbox) ValidatePath(path string, write bool) error {
	if !write {
		return nil
	}
	return validateWritePath(path, b.opts)
}

func (b *bwrapSandbox) Available() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

func (b *bwrapSandbox) Name() string { return "bwrap" }

// buildArgs generates bwrap command-line arguments.
func (b *bwrapSandbox) buildArgs() []string {
	args := []string{
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
	}

	if b.opts.WorkDir != "" {
		abs, _ := filepath.Abs(b.opts.WorkDir)
		if abs != "" {
			args = append(args, "--bind", abs, abs)
		}
	}

	for _, dir := range b.opts.AdditionalDirs {
		abs, _ := filepath.Abs(dir)
		if abs != "" {
			args = append(args, "--bind", abs, abs)
		}
	}

	if !b.opts.AllowNetwork {
		args = append(args, "--unshare-net")
	}

	return args
}
