// ABOUTME: Tests for BackgroundManager: add, remove, list, count, limits, mark done
// ABOUTME: Validates concurrent-safe task lifecycle and max-5 enforcement

package btea

import (
	"testing"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBackgroundManager_AddAndList(t *testing.T) {
	mgr := NewBackgroundManager(nil)

	task := &BackgroundTask{
		ID:        "bg-001",
		Prompt:    "do something",
		StartedAt: time.Now(),
		Status:    BGRunning,
	}
	err := mgr.Add(task)
	if err != nil {
		t.Fatalf("Add() error = %v; want nil", err)
	}

	tasks := mgr.List()
	if len(tasks) != 1 {
		t.Fatalf("List() len = %d; want 1", len(tasks))
	}
	if tasks[0].ID != "bg-001" {
		t.Errorf("List()[0].ID = %q; want %q", tasks[0].ID, "bg-001")
	}
}

func TestBackgroundManager_Get(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	task := &BackgroundTask{
		ID:        "bg-002",
		Prompt:    "test prompt",
		StartedAt: time.Now(),
		Status:    BGRunning,
	}
	_ = mgr.Add(task)

	got := mgr.Get("bg-002")
	if got == nil {
		t.Fatal("Get() = nil; want task")
	}
	if got.Prompt != "test prompt" {
		t.Errorf("Get().Prompt = %q; want %q", got.Prompt, "test prompt")
	}

	if mgr.Get("nonexistent") != nil {
		t.Error("Get(nonexistent) != nil; want nil")
	}
}

func TestBackgroundManager_Remove(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	task := &BackgroundTask{
		ID:        "bg-003",
		Prompt:    "removable",
		StartedAt: time.Now(),
		Status:    BGDone,
	}
	_ = mgr.Add(task)

	mgr.Remove("bg-003")
	if mgr.Count() != 0 {
		t.Errorf("Count() = %d after Remove; want 0", mgr.Count())
	}
}

func TestBackgroundManager_Count(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	if mgr.Count() != 0 {
		t.Errorf("Count() = %d; want 0", mgr.Count())
	}

	for i := range 3 {
		_ = mgr.Add(&BackgroundTask{
			ID:        "bg-" + string(rune('a'+i)),
			Prompt:    "task",
			StartedAt: time.Now(),
			Status:    BGRunning,
		})
	}
	if mgr.Count() != 3 {
		t.Errorf("Count() = %d; want 3", mgr.Count())
	}
}

func TestBackgroundManager_RunningCount(t *testing.T) {
	mgr := NewBackgroundManager(nil)

	_ = mgr.Add(&BackgroundTask{ID: "bg-r1", Prompt: "a", StartedAt: time.Now(), Status: BGRunning})
	_ = mgr.Add(&BackgroundTask{ID: "bg-r2", Prompt: "b", StartedAt: time.Now(), Status: BGDone})
	_ = mgr.Add(&BackgroundTask{ID: "bg-r3", Prompt: "c", StartedAt: time.Now(), Status: BGRunning})

	if mgr.RunningCount() != 2 {
		t.Errorf("RunningCount() = %d; want 2", mgr.RunningCount())
	}
}

func TestBackgroundManager_MaxLimit(t *testing.T) {
	mgr := NewBackgroundManager(nil)

	for i := range MaxBackgroundTasks {
		err := mgr.Add(&BackgroundTask{
			ID:        "bg-" + string(rune('0'+i)),
			Prompt:    "task",
			StartedAt: time.Now(),
			Status:    BGRunning,
		})
		if err != nil {
			t.Fatalf("Add(%d) error = %v; want nil", i, err)
		}
	}

	err := mgr.Add(&BackgroundTask{
		ID:        "bg-overflow",
		Prompt:    "over limit",
		StartedAt: time.Now(),
		Status:    BGRunning,
	})
	if err == nil {
		t.Fatal("Add() error = nil; want max limit error")
	}
}

func TestBackgroundManager_MarkDone(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	_ = mgr.Add(&BackgroundTask{
		ID:        "bg-mark",
		Prompt:    "mark me done",
		StartedAt: time.Now(),
		Status:    BGRunning,
	})

	msgs := []ai.Message{{Role: ai.RoleAssistant}}
	mgr.MarkDone("bg-mark", msgs, nil)

	task := mgr.Get("bg-mark")
	if task.Status != BGDone {
		t.Errorf("Status = %d; want BGDone (%d)", task.Status, BGDone)
	}
	if len(task.Messages) != 1 {
		t.Errorf("Messages len = %d; want 1", len(task.Messages))
	}
}

func TestBackgroundManager_MarkDoneFailed(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	_ = mgr.Add(&BackgroundTask{
		ID:        "bg-fail",
		Prompt:    "will fail",
		StartedAt: time.Now(),
		Status:    BGRunning,
	})

	testErr := &bgTestError{"agent error"}
	mgr.MarkDone("bg-fail", nil, testErr)

	task := mgr.Get("bg-fail")
	if task.Status != BGFailed {
		t.Errorf("Status = %d; want BGFailed (%d)", task.Status, BGFailed)
	}
	if task.Err == nil {
		t.Error("Err = nil; want error")
	}
}

func TestBackgroundStatus_String(t *testing.T) {
	tests := []struct {
		status BackgroundStatus
		want   string
	}{
		{BGRunning, "running"},
		{BGDone, "done"},
		{BGFailed, "failed"},
		{BackgroundStatus(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("BackgroundStatus(%d).String() = %q; want %q", tt.status, got, tt.want)
		}
	}
}

func TestBackgroundManager_ListReturnsValueCopies(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	_ = mgr.Add(&BackgroundTask{
		ID:        "bg-snap",
		Prompt:    "snapshot test",
		StartedAt: time.Now(),
		Status:    BGRunning,
	})

	tasks := mgr.List()
	if len(tasks) != 1 {
		t.Fatalf("List() len = %d; want 1", len(tasks))
	}

	// Mutating the returned value must NOT affect the manager's internal state.
	tasks[0].Status = BGDone
	tasks[0].Prompt = "mutated"

	got := mgr.Get("bg-snap")
	if got.Status != BGRunning {
		t.Errorf("internal Status changed to %v after mutating List() result; want BGRunning", got.Status)
	}
	if got.Prompt != "snapshot test" {
		t.Errorf("internal Prompt changed to %q after mutating List() result; want %q", got.Prompt, "snapshot test")
	}
}

func TestBackgroundManager_GetReturnsCopy(t *testing.T) {
	mgr := NewBackgroundManager(nil)
	_ = mgr.Add(&BackgroundTask{
		ID:        "bg-copy",
		Prompt:    "copy test",
		StartedAt: time.Now(),
		Status:    BGRunning,
	})

	got := mgr.Get("bg-copy")
	got.Status = BGFailed
	got.Prompt = "tampered"

	original := mgr.Get("bg-copy")
	if original.Status != BGRunning {
		t.Errorf("internal Status changed to %v after mutating Get() result; want BGRunning", original.Status)
	}
	if original.Prompt != "copy test" {
		t.Errorf("internal Prompt changed to %q after mutating Get() result; want %q", original.Prompt, "copy test")
	}
}

func TestBackgroundTask_Snapshot(t *testing.T) {
	task := &BackgroundTask{
		ID:        "bg-orig",
		Prompt:    "original",
		StartedAt: time.Now(),
		Status:    BGRunning,
		Messages:  []ai.Message{{Role: ai.RoleAssistant}},
	}

	snap := task.Snapshot()
	snap.Status = BGDone
	snap.Prompt = "changed"
	snap.Messages[0].Role = ai.RoleUser

	if task.Status != BGRunning {
		t.Errorf("original Status changed after Snapshot mutation")
	}
	if task.Prompt != "original" {
		t.Errorf("original Prompt changed after Snapshot mutation")
	}
	// Messages slice is copied, so modifying snap.Messages should not affect original.
	if task.Messages[0].Role != ai.RoleAssistant {
		t.Errorf("original Messages changed after Snapshot mutation")
	}
}

type bgTestError struct{ msg string }

func (e *bgTestError) Error() string { return e.msg }
