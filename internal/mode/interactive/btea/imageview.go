// ABOUTME: Bubble Tea image view model for rendering images in tool call boxes
// ABOUTME: Delegates to pkg/tui/image.Render; renders eagerly at construction time

package btea

import (
	"strings"

	img "github.com/mauromedda/pi-coding-agent-go/pkg/tui/image"
)

// ImageViewModel renders an image in the TUI.
// Output is computed eagerly in the constructor; View() is a pure accessor.
// This avoids mutation issues with Bubble Tea's value-copy model.
type ImageViewModel struct {
	data     []byte
	mimeType string
	width    int
	output   string // Pre-rendered output
}

// NewImageViewModel creates an image view model and renders the image immediately.
func NewImageViewModel(data []byte, mimeType string, width int) ImageViewModel {
	m := ImageViewModel{
		data:     data,
		mimeType: mimeType,
		width:    width,
	}
	if len(data) == 0 || width <= 0 {
		return m
	}

	lines, err := img.Render(data, mimeType, width)
	if err != nil {
		m.output = img.ImagePlaceholder(data)
	} else {
		m.output = strings.Join(lines, "\n")
	}
	return m
}

// View returns the pre-rendered image string.
func (m ImageViewModel) View() string {
	return m.output
}
