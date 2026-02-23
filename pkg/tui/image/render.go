// ABOUTME: High-level image rendering dispatcher
// ABOUTME: Routes to Kitty, iTerm2, or half-block based on detected terminal capability

package image

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/png"

	// Register decoders for standard formats.
	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/webp"
)

const (
	// MaxFileSize is the maximum image file size accepted (4.5 MB).
	MaxFileSize = 4_500_000
	// MaxDimension is the maximum width or height in pixels.
	MaxDimension = 2000
)

// Render produces terminal-ready output lines for the given image data.
// It detects the terminal protocol and dispatches accordingly:
//   - Kitty: ensures PNG, encodes with chunked protocol (single line)
//   - iTerm2: encodes with OSC 1337 (single line)
//   - None: decodes image, renders as half-block ANSI art (multiple lines)
//
// On decode error for half-block, returns a text placeholder instead of an error.
func Render(data []byte, mimeType string, maxCols int) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image data")
	}

	cap := Detect()

	switch cap.Images {
	case ProtoKitty:
		return renderKitty(data, maxCols)
	case ProtoITerm2:
		return renderITerm2(data, maxCols)
	default:
		return renderHalfBlockFromBytes(data, mimeType, maxCols)
	}
}

// renderKitty converts the image to PNG (if needed) and encodes for Kitty.
func renderKitty(data []byte, maxCols int) ([]string, error) {
	pngData, err := ensurePNG(data)
	if err != nil {
		return nil, fmt.Errorf("preparing PNG for Kitty: %w", err)
	}
	dim, _ := GetDimensions(pngData)
	if dim.Width < 1 || dim.Height < 1 {
		dim = Dimensions{Width: maxCols, Height: maxCols}
	}

	// Scale proportionally to fit maxCols; cells have ~1:2 pixel aspect ratio
	scale := 1.0
	if dim.Width > maxCols {
		scale = float64(maxCols) / float64(dim.Width)
	}
	cols := int(float64(dim.Width) * scale)
	rows := int(float64(dim.Height) * scale / 2.0)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return []string{EncodeKitty(pngData, cols, rows)}, nil
}

// renderITerm2 encodes the image data for iTerm2 inline display.
func renderITerm2(data []byte, maxCols int) ([]string, error) {
	width := fmt.Sprintf("%d", maxCols)
	return []string{EncodeITerm2(data, width)}, nil
}

// renderHalfBlockFromBytes decodes the image and renders with half-block chars.
// Returns a placeholder on decode error instead of failing.
func renderHalfBlockFromBytes(data []byte, mimeType string, maxCols int) ([]string, error) {
	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		// Graceful fallback: text placeholder
		dim, _ := GetDimensions(data)
		placeholder := fmt.Sprintf("[Image: %s %dx%d]", mimeType, dim.Width, dim.Height)
		return []string{placeholder}, nil
	}

	lines := RenderHalfBlock(img, maxCols)
	if len(lines) == 0 {
		return []string{ImagePlaceholder(data)}, nil
	}
	return lines, nil
}

// ensurePNG re-encodes the image as PNG if it isn't already.
func ensurePNG(data []byte) ([]byte, error) {
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return data, nil
	}
	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding for PNG conversion: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}
