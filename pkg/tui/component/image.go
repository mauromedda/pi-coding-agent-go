// ABOUTME: Image component for rendering images via terminal graphics protocols
// ABOUTME: Supports Kitty and iTerm2 protocols with chunked base64 transmission

package component

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// ImageProtocol identifies the terminal image protocol to use.
type ImageProtocol int

const (
	// ProtocolKitty uses the Kitty terminal graphics protocol.
	ProtocolKitty ImageProtocol = iota
	// ProtocolITerm2 uses the iTerm2 inline images protocol.
	ProtocolITerm2
)

// String returns the protocol name.
func (p ImageProtocol) String() string {
	switch p {
	case ProtocolKitty:
		return "kitty"
	case ProtocolITerm2:
		return "iterm2"
	default:
		return "unknown"
	}
}

const kittyChunkSize = 4096

// Image renders image data using terminal graphics protocols.
type Image struct {
	data     []byte
	protocol ImageProtocol
	dirty    bool
	cached   []string
}

// NewImage creates an Image component with the given raw image data.
func NewImage(data []byte) *Image {
	return &Image{data: data, dirty: true}
}

// SetProtocol sets the image rendering protocol.
func (img *Image) SetProtocol(p ImageProtocol) {
	img.protocol = p
	img.dirty = true
}

// SetData replaces the image data.
func (img *Image) SetData(data []byte) {
	img.data = data
	img.dirty = true
}

// Render writes the image escape sequences into the buffer.
func (img *Image) Render(out *tui.RenderBuffer, _ int) {
	if len(img.data) == 0 {
		return
	}
	if img.dirty {
		img.cached = img.renderLines()
		img.dirty = false
	}
	out.WriteLines(img.cached)
}

// Invalidate forces re-rendering on next Render call.
func (img *Image) Invalidate() {
	img.dirty = true
}

// DetectProtocol returns the appropriate protocol based on the terminal program name.
func DetectProtocol(termProg string) ImageProtocol {
	lower := strings.ToLower(termProg)
	if strings.Contains(lower, "iterm") {
		return ProtocolITerm2
	}
	return ProtocolKitty
}

func (img *Image) renderLines() []string {
	switch img.protocol {
	case ProtocolITerm2:
		return img.renderITerm2()
	default:
		return img.renderKitty()
	}
}

func (img *Image) renderKitty() []string {
	encoded := base64.StdEncoding.EncodeToString(img.data)

	if len(encoded) <= kittyChunkSize {
		line := fmt.Sprintf("\x1b_Gf=100,a=T,m=0;%s\x1b\\", encoded)
		return []string{line}
	}

	var lines []string
	for i := 0; i < len(encoded); i += kittyChunkSize {
		end := i + kittyChunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		more := 1
		if end == len(encoded) {
			more = 0
		}
		if i == 0 {
			lines = append(lines, fmt.Sprintf("\x1b_Gf=100,a=T,m=%d;%s\x1b\\", more, chunk))
		} else {
			lines = append(lines, fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, chunk))
		}
	}
	return lines
}

func (img *Image) renderITerm2() []string {
	encoded := base64.StdEncoding.EncodeToString(img.data)
	line := fmt.Sprintf("\x1b]1337;File=size=%d;inline=1:%s\a", len(img.data), encoded)
	return []string{line}
}
