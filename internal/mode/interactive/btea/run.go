// ABOUTME: Entry point for the Bubble Tea interactive TUI
// ABOUTME: Creates the tea.Program, injects program reference, and blocks until exit

package btea

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the Bubble Tea interactive app. Blocks until the user exits.
// The deps struct provides all external dependencies (provider, model, tools, etc.).
func Run(deps AppDeps) error {
	m := NewAppModel(deps)

	p := tea.NewProgram(
		m,
		tea.WithOutput(os.Stderr),
	)

	// Inject the program reference into the shared state.
	// This is safe because NewAppModel allocates sh as a pointer,
	// and tea.NewProgram copies the model value but shares the pointer.
	m.sh.program = p

	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("bubble tea: %w", err)
	}
	return nil
}
