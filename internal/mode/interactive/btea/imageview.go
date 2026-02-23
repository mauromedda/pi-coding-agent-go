// ABOUTME: Bubble Tea image view model for rendering images in tool call boxes
// ABOUTME: Delegates to pkg/tui/image.Render; caches output lines

package btea

import (
	"strings"

	img "github.com/mauromedda/pi-coding-agent-go/pkg/tui/image"
)

// ImageViewModel renders an image in the TUI.
// The rendered output is cached after the first call to View().
type ImageViewModel struct {
	data     []byte
	mimeType string
	width    int
	lines    []string
	rendered bool
}

// NewImageViewModel creates an image view model for the given image data.
func NewImageViewModel(data []byte, mimeType string, width int) ImageViewModel {
	return ImageViewModel{
		data:     data,
		mimeType: mimeType,
		width:    width,
	}
}

// View returns the rendered image as a string.
// Uses the terminal's detected image protocol or half-block fallback.
func (m *ImageViewModel) View() string {
	if len(m.data) == 0 {
		return ""
	}

	if !m.rendered {
		lines, err := img.Render(m.data, m.mimeType, m.width)
		if err != nil {
			m.lines = []string{img.ImagePlaceholder(m.data)}
		} else {
			m.lines = lines
		}
		m.rendered = true
	}

	return strings.Join(m.lines, "\n")
}
