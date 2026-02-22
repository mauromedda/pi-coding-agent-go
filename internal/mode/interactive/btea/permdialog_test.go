// ABOUTME: Tests for PermDialogModel overlay: permission request, reply via channel
// ABOUTME: Verifies View rendering, key handling (y/a/n/esc), and DismissOverlayMsg

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: PermDialogModel must satisfy tea.Model.
var _ tea.Model = PermDialogModel{}

// Compile-time check: DismissOverlayMsg must satisfy tea.Msg.
var _ tea.Msg = DismissOverlayMsg{}

func TestPermDialogModel_Init(t *testing.T) {
	ch := make(chan<- PermissionReply, 1)
	m := NewPermDialogModel("Bash", nil, ch)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestPermDialogModel_ViewContainsToolName(t *testing.T) {
	ch := make(chan<- PermissionReply, 1)
	m := NewPermDialogModel("Bash", map[string]any{"command": "ls"}, ch)
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "Permission Required") {
		t.Error("View() missing 'Permission Required' header")
	}
	if !strings.Contains(view, "Bash") {
		t.Error("View() missing tool name 'Bash'")
	}
}

func TestPermDialogModel_ViewShowsArgs(t *testing.T) {
	ch := make(chan<- PermissionReply, 1)
	args := map[string]any{"command": "ls -la", "path": "/tmp"}
	m := NewPermDialogModel("Bash", args, ch)
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "command=ls -la") {
		t.Errorf("View() missing arg 'command=ls -la'; got:\n%s", view)
	}
}

func TestPermDialogModel_ViewShowsOptions(t *testing.T) {
	ch := make(chan<- PermissionReply, 1)
	m := NewPermDialogModel("Read", nil, ch)
	m.width = 80
	view := m.View()

	for _, want := range []string{"[y] Allow", "[a] Always", "[n] Deny"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing option %q", want)
		}
	}
}

func TestPermDialogModel_KeyY(t *testing.T) {
	replyCh := make(chan PermissionReply, 1)
	m := NewPermDialogModel("Bash", nil, replyCh)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	_ = updated

	if cmd == nil {
		t.Fatal("Update('y') returned nil cmd; want dismissOverlayCmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() returned %T; want DismissOverlayMsg", msg)
	}

	select {
	case reply := <-replyCh:
		if !reply.Allowed {
			t.Error("reply.Allowed = false; want true")
		}
		if reply.Always {
			t.Error("reply.Always = true; want false")
		}
	default:
		t.Fatal("no reply sent on channel after 'y' key")
	}
}

func TestPermDialogModel_KeyA(t *testing.T) {
	replyCh := make(chan PermissionReply, 1)
	m := NewPermDialogModel("Bash", nil, replyCh)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_ = updated

	if cmd == nil {
		t.Fatal("Update('a') returned nil cmd; want dismissOverlayCmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() returned %T; want DismissOverlayMsg", msg)
	}

	select {
	case reply := <-replyCh:
		if !reply.Allowed {
			t.Error("reply.Allowed = false; want true")
		}
		if !reply.Always {
			t.Error("reply.Always = false; want true")
		}
	default:
		t.Fatal("no reply sent on channel after 'a' key")
	}
}

func TestPermDialogModel_KeyN(t *testing.T) {
	replyCh := make(chan PermissionReply, 1)
	m := NewPermDialogModel("Bash", nil, replyCh)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	_ = updated

	if cmd == nil {
		t.Fatal("Update('n') returned nil cmd; want dismissOverlayCmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() returned %T; want DismissOverlayMsg", msg)
	}

	select {
	case reply := <-replyCh:
		if reply.Allowed {
			t.Error("reply.Allowed = true; want false")
		}
	default:
		t.Fatal("no reply sent on channel after 'n' key")
	}
}

func TestPermDialogModel_KeyEsc(t *testing.T) {
	replyCh := make(chan PermissionReply, 1)
	m := NewPermDialogModel("Bash", nil, replyCh)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updated

	if cmd == nil {
		t.Fatal("Update(esc) returned nil cmd; want dismissOverlayCmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() returned %T; want DismissOverlayMsg", msg)
	}

	select {
	case reply := <-replyCh:
		if reply.Allowed {
			t.Error("reply.Allowed = true; want false on esc")
		}
	default:
		t.Fatal("no reply sent on channel after esc key")
	}
}

func TestPermDialogModel_UnrelatedKeyNoReply(t *testing.T) {
	replyCh := make(chan PermissionReply, 1)
	m := NewPermDialogModel("Bash", nil, replyCh)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Errorf("Update('x') returned non-nil cmd; want nil for unrelated key")
	}

	select {
	case <-replyCh:
		t.Fatal("unexpected reply on channel for unrelated key 'x'")
	default:
		// expected: no reply
	}
}

func TestPermDialogModel_ViewNoArgs(t *testing.T) {
	ch := make(chan<- PermissionReply, 1)
	m := NewPermDialogModel("Read", nil, ch)
	m.width = 80
	view := m.View()

	// Should still render without args
	if !strings.Contains(view, "Read") {
		t.Error("View() missing tool name 'Read' with nil args")
	}
}

func TestDismissOverlayCmd(t *testing.T) {
	msg := dismissOverlayCmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("dismissOverlayCmd() returned %T; want DismissOverlayMsg", msg)
	}
}
