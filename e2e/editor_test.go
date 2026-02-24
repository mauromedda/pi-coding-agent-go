// ABOUTME: E2E tests for editor interactions: Ctrl+C quit, Ctrl+D quit, Ctrl+L clear
// ABOUTME: Tests keyboard shortcuts and editor behavior through the real binary PTY

package e2e

import (
	"testing"
	"time"
)

func TestEditor_CtrlC_ClearsThenExits(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e tests skipped in short mode")
	}

	s := startPi(t)
	defer s.close()

	s.expectStringTimeout(t, "pi-go", 5*time.Second)

	// First Ctrl+C clears the conversation.
	s.sendCtrl(t, 'c')
	time.Sleep(200 * time.Millisecond)

	// Welcome screen should still be visible after clear.
	s.expectStringTimeout(t, "pi-go", 5*time.Second)

	// Second Ctrl+C within 1s exits the application.
	s.sendCtrl(t, 'c')

	s.waitExit(t, 5*time.Second)
}

func TestEditor_CtrlD_ExitsWhenEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e tests skipped in short mode")
	}

	s := startPi(t)
	defer s.close()

	s.expectStringTimeout(t, "pi-go", 5*time.Second)

	// Ctrl+D on empty input should exit.
	s.sendCtrl(t, 'd')

	s.waitExit(t, 5*time.Second)
}

func TestEditor_CtrlL_ClearsScreen(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e tests skipped in short mode")
	}

	s := startPi(t)
	defer s.close()

	s.expectStringTimeout(t, "pi-go", 5*time.Second)

	// Submit a /help command to add content.
	submitCommand(t, s, "help")
	s.expectStringTimeout(t, "commands", 10*time.Second)

	// Ctrl+L clears the content area and shows fresh welcome.
	s.sendCtrl(t, 'l')
	time.Sleep(500 * time.Millisecond)

	// After clear, a fresh welcome model is added back.
	// The pi-go text should still be visible (from the new welcome).
	s.expectStringTimeout(t, "pi-go", 5*time.Second)
}

func TestEditor_ShiftTab_TogglesMode(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e tests skipped in short mode")
	}

	s := startPi(t)
	defer s.close()

	s.expectStringTimeout(t, "pi-go", 5*time.Second)

	// Send Shift+Tab.
	s.sendShiftTab(t)

	// Footer should show Plan mode.
	s.expectStringTimeout(t, "Plan", 5*time.Second)
}

func TestEditor_SlashOpensCommandPalette(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e tests skipped in short mode")
	}

	s := startPi(t)
	defer s.close()

	s.expectStringTimeout(t, "pi-go", 5*time.Second)

	// Typing / should open the command palette overlay.
	s.send(t, "/")
	time.Sleep(500 * time.Millisecond)

	// Palette shows commands. Type "hel" to filter.
	s.send(t, "hel")
	time.Sleep(300 * time.Millisecond)

	// Should see /help in the palette.
	s.expectStringTimeout(t, "help", 3*time.Second)

	// Dismiss palette with Escape.
	s.sendEscape(t)
	time.Sleep(300 * time.Millisecond)
}
