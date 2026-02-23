// ABOUTME: ANSI half-block character fallback renderer for terminals without image protocols
// ABOUTME: Uses ▄ with fg/bg true-color escapes to double vertical resolution

package image

import (
	"fmt"
	goimage "image"
	"strings"

	"golang.org/x/image/draw"
)

// RenderHalfBlock converts an image to ANSI art using the lower-half block character (▄).
// For every 2 rows of pixels: background = top pixel color, foreground = bottom pixel color.
// The image is scaled to maxCols width preserving aspect ratio.
func RenderHalfBlock(img goimage.Image, maxCols int) []string {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 || srcH == 0 || maxCols == 0 {
		return nil
	}

	// Scale to fit maxCols width
	targetW := srcW
	targetH := srcH
	if targetW > maxCols {
		targetH = targetH * maxCols / targetW
		targetW = maxCols
	}
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}

	// Resize if needed
	var scaled goimage.Image
	if targetW != srcW || targetH != srcH {
		dst := goimage.NewRGBA(goimage.Rect(0, 0, targetW, targetH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		scaled = dst
	} else {
		scaled = img
	}

	// Render pairs of rows using ▄
	var lines []string
	for y := 0; y < targetH; y += 2 {
		var b strings.Builder
		for x := range targetW {
			// Top pixel → background color
			topR, topG, topB := rgbAt(scaled, x, y)

			// Bottom pixel → foreground color (black if out of bounds)
			var botR, botG, botB uint8
			if y+1 < targetH {
				botR, botG, botB = rgbAt(scaled, x, y+1)
			}

			fmt.Fprintf(&b, "\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm▄",
				topR, topG, topB, botR, botG, botB)
		}
		b.WriteString("\x1b[0m")
		lines = append(lines, b.String())
	}

	return lines
}

// rgbAt extracts the 8-bit RGB components of the pixel at (x, y).
func rgbAt(img goimage.Image, x, y int) (uint8, uint8, uint8) {
	r, g, b, _ := img.At(x, y).RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

