// ABOUTME: Opens content in $EDITOR for editing (Ctrl+G integration)
// ABOUTME: Writes to temp file, launches editor, reads back edited content

package ide

import (
	"fmt"
	"os"
	"os/exec"
)

// OpenInEditor writes content to a temp file, launches $EDITOR,
// blocks until the editor exits, and returns the edited content.
func OpenInEditor(content string) (string, error) {
	editor := getEditor()

	tmpFile, err := os.CreateTemp("", "pi-go-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running editor %s: %w", editor, err)
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}

	return string(edited), nil
}

func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}
	return "vi"
}
