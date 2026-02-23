// ABOUTME: Git-based checkpoint/rewind before tool execution
// ABOUTME: Uses git stash create for snapshots; supports rewind to any checkpoint

package ide

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Checkpoint captures the working tree state before a tool execution.
type Checkpoint struct {
	Ref         string    // Git stash ref or checkpoint directory path
	Timestamp   time.Time
	ToolName    string
	ToolArgs    string
	Name        string // Optional human-readable name
	Description string // Optional description of what this checkpoint captures
	Mode        string // Permission mode at time of checkpoint (e.g., "plan", "execute")
}

// CheckpointStack manages a stack of checkpoints for the current session.
type CheckpointStack struct {
	mu    sync.Mutex
	stack []Checkpoint
	cwd   string
}

// NewCheckpointStack creates a stack for the given working directory.
func NewCheckpointStack(cwd string) *CheckpointStack {
	return &CheckpointStack{cwd: cwd}
}

// Save creates a checkpoint of the current working tree.
func (s *CheckpointStack) Save(toolName, toolArgs string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref, err := gitStashCreate(s.cwd)
	if err != nil {
		return fmt.Errorf("creating checkpoint: %w", err)
	}
	if ref == "" {
		return nil // No changes to checkpoint
	}

	s.stack = append(s.stack, Checkpoint{
		Ref:       ref,
		Timestamp: time.Now(),
		ToolName:  toolName,
		ToolArgs:  toolArgs,
	})
	return nil
}

// SaveNamed creates a checkpoint with a human-readable name and description.
func (s *CheckpointStack) SaveNamed(name, description, toolName, toolArgs, mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref, err := gitStashCreate(s.cwd)
	if err != nil {
		return fmt.Errorf("creating checkpoint: %w", err)
	}
	if ref == "" {
		return nil // No changes to checkpoint
	}

	s.stack = append(s.stack, Checkpoint{
		Ref:         ref,
		Timestamp:   time.Now(),
		ToolName:    toolName,
		ToolArgs:    toolArgs,
		Name:        name,
		Description: description,
		Mode:        mode,
	})
	return nil
}

// Rewind restores the most recent checkpoint.
func (s *CheckpointStack) Rewind() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.stack) == 0 {
		return fmt.Errorf("no checkpoints to rewind")
	}

	last := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]

	return gitStashApply(s.cwd, last.Ref)
}

// RewindTo restores the nth checkpoint from the top (0 = most recent).
func (s *CheckpointStack) RewindTo(n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n < 0 || n >= len(s.stack) {
		return fmt.Errorf("checkpoint index %d out of range (have %d)", n, len(s.stack))
	}

	idx := len(s.stack) - 1 - n
	cp := s.stack[idx]
	s.stack = s.stack[:idx]

	return gitStashApply(s.cwd, cp.Ref)
}

// List returns all checkpoints, newest first.
func (s *CheckpointStack) List() []Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Checkpoint, len(s.stack))
	for i, cp := range s.stack {
		result[len(s.stack)-1-i] = cp
	}
	return result
}

func gitStashCreate(cwd string) (string, error) {
	cmd := exec.Command("git", "stash", "create")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git stash create: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitStashApply(cwd, ref string) error {
	cmd := exec.Command("git", "stash", "apply", ref)
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git stash apply %s: %s: %w", ref, string(out), err)
	}
	return nil
}
