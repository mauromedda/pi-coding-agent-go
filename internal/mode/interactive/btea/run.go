// ABOUTME: Entry point for the Bubble Tea interactive TUI
// ABOUTME: Creates the tea.Program, injects program reference, and blocks until exit

package btea

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mauromedda/pi-coding-agent-go/internal/git"
)

// Run starts the Bubble Tea interactive app. Blocks until the user exits.
// The deps struct provides all external dependencies (provider, model, tools, etc.).
func Run(deps AppDeps) error {
	// Pre-set dark background to prevent Lipgloss from sending OSC 10/11
	// terminal queries at runtime. Late-arriving responses leak into the
	// Bubble Tea input parser and appear as garbled text in the editor.
	lipgloss.SetHasDarkBackground(true)

	m := NewAppModel(deps)

	p := tea.NewProgram(
		m,
		tea.WithOutput(os.Stderr),
	)

	// Inject the program reference into the shared state.
	// Safe because NewAppModel allocates sh as a pointer and tea.NewProgram
	// copies the model value but shares the pointer. The event loop has
	// not started yet (Run hasn't been called), so no concurrent access.
	m.sh.program = p
	m.sh.bgManager = NewBackgroundManager(p)
	defer m.sh.cancel() // cancel root context when program exits

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("bubble tea: %w", err)
	}

	// Handle worktree cleanup after the program exits.
	if deps.WorktreeSession != nil {
		if appModel, ok := finalModel.(AppModel); ok {
			handleWorktreeExit(deps.WorktreeSession, appModel.worktreeExitAction)
		}
	}

	return nil
}

// handleWorktreeExit performs the chosen worktree cleanup action after the TUI exits.
func handleWorktreeExit(sw *git.SessionWorktree, action WorktreeExitAction) {
	switch action {
	case WorktreeActionMerge:
		if err := sw.Merge(); err != nil {
			fmt.Fprintf(os.Stderr, "worktree merge failed: %v\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "Merged worktree branch %q into %q\n", sw.Info.Branch, sw.OrigBranch)
	case WorktreeActionKeep:
		if err := sw.Keep(); err != nil {
			fmt.Fprintf(os.Stderr, "worktree keep failed: %v\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "Kept worktree at %s (branch: %s)\n", sw.Info.Path, sw.Info.Branch)
	case WorktreeActionDiscard:
		if err := sw.Discard(); err != nil {
			fmt.Fprintf(os.Stderr, "worktree discard failed: %v\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "Discarded worktree and branch %q\n", sw.Info.Branch)
	}
}
