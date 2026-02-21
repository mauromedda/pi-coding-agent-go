// ABOUTME: Tests for the SGR state machine tracker
// ABOUTME: Covers bold, color, reset, and restore sequence generation

package ansitrack

import "testing"

func TestTracker_Bold(t *testing.T) {
	t.Parallel()

	var tr Tracker
	tr.Process("\x1b[1m")

	if !tr.IsActive() {
		t.Error("expected active after bold")
	}
	got := tr.Restore()
	if got != "\x1b[1m" {
		t.Errorf("Restore() = %q, want %q", got, "\x1b[1m")
	}
}

func TestTracker_FGColor(t *testing.T) {
	t.Parallel()

	var tr Tracker
	tr.Process("\x1b[31m")

	got := tr.Restore()
	if got != "\x1b[31m" {
		t.Errorf("Restore() = %q, want %q", got, "\x1b[31m")
	}
}

func TestTracker_Reset(t *testing.T) {
	t.Parallel()

	var tr Tracker
	tr.Process("\x1b[1m")
	tr.Process("\x1b[31m")
	tr.Process("\x1b[0m")

	if tr.IsActive() {
		t.Error("expected inactive after reset")
	}
	if got := tr.Restore(); got != "" {
		t.Errorf("Restore() = %q, want empty", got)
	}
}

func TestTracker_CombinedSequence(t *testing.T) {
	t.Parallel()

	var tr Tracker
	tr.Process("\x1b[1;31m")

	if !tr.bold || tr.fg != "31" {
		t.Errorf("expected bold+red, got bold=%v fg=%q", tr.bold, tr.fg)
	}
}
