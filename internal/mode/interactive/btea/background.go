// ABOUTME: BackgroundManager tracks detached agent tasks running silently
// ABOUTME: Thread-safe via sync.Mutex; enforces MaxBackgroundTasks limit of 5

package btea

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// MaxBackgroundTasks is the maximum number of concurrent background tasks.
const MaxBackgroundTasks = 5

// BackgroundStatus represents the execution state of a background task.
type BackgroundStatus int

const (
	// BGRunning means the agent is still executing.
	BGRunning BackgroundStatus = iota
	// BGDone means the agent completed successfully.
	BGDone
	// BGFailed means the agent terminated with an error.
	BGFailed
)

// String returns a human-readable label for the status.
func (s BackgroundStatus) String() string {
	switch s {
	case BGRunning:
		return "running"
	case BGDone:
		return "done"
	case BGFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// BackgroundTask holds metadata and results for a detached agent run.
type BackgroundTask struct {
	ID        string
	Prompt    string
	StartedAt time.Time
	Status    BackgroundStatus
	Messages  []ai.Message // populated on completion
	Err       error
	Cancel    context.CancelFunc
}

// Snapshot returns a shallow copy of the task with a copied Messages slice.
// Safe to read without holding the manager lock.
func (t *BackgroundTask) Snapshot() BackgroundTask {
	cp := *t
	if t.Messages != nil {
		cp.Messages = make([]ai.Message, len(t.Messages))
		copy(cp.Messages, t.Messages)
	}
	return cp
}

// BackgroundManager manages the set of background tasks.
// All methods are safe for concurrent use.
type BackgroundManager struct {
	mu      sync.Mutex
	tasks   map[string]*BackgroundTask
	program ProgramSender
}

// NewBackgroundManager creates a BackgroundManager wired to the given program
// for sending completion notifications.
func NewBackgroundManager(program ProgramSender) *BackgroundManager {
	return &BackgroundManager{
		tasks:   make(map[string]*BackgroundTask),
		program: program,
	}
}

// Add registers a new background task. Returns an error if the limit is reached.
func (m *BackgroundManager) Add(task *BackgroundTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.tasks) >= MaxBackgroundTasks {
		return fmt.Errorf("background task limit reached (%d)", MaxBackgroundTasks)
	}
	m.tasks[task.ID] = task
	return nil
}

// Remove deletes a task by ID.
func (m *BackgroundManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, id)
}

// Get returns a snapshot copy of the task with the given ID, or nil if not found.
func (m *BackgroundManager) Get(id string) *BackgroundTask {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return nil
	}
	snap := t.Snapshot()
	return &snap
}

// List returns snapshot copies of all tasks. Safe to read after the lock is released.
func (m *BackgroundManager) List() []BackgroundTask {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]BackgroundTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		result = append(result, t.Snapshot())
	}
	return result
}

// Count returns the total number of background tasks.
func (m *BackgroundManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tasks)
}

// RunningCount returns the number of tasks with status BGRunning.
func (m *BackgroundManager) RunningCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	n := 0
	for _, t := range m.tasks {
		if t.Status == BGRunning {
			n++
		}
	}
	return n
}

// MarkDone updates a task's status to BGDone (or BGFailed if err != nil)
// and stores the result messages.
func (m *BackgroundManager) MarkDone(id string, messages []ai.Message, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return
	}
	t.Messages = messages
	t.Err = err
	if err != nil {
		t.Status = BGFailed
	} else {
		t.Status = BGDone
	}
}
