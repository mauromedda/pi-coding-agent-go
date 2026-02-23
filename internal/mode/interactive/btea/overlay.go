// ABOUTME: overlayRender composites an overlay box centered on a background terminal view
// ABOUTME: Splices overlay lines into background at vertical/horizontal center, preserving non-overlay rows

package btea

import (
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// overlayRender composites overlay text centered on top of background text.
// Background lines outside the overlay region are preserved; overlay lines
// are spliced in at the vertical and horizontal center.
func overlayRender(background, overlay string, termWidth, termHeight int) string {
	bgLines := strings.Split(background, "\n")

	// Pad or trim background to exactly termHeight lines
	for len(bgLines) < termHeight {
		bgLines = append(bgLines, "")
	}
	if len(bgLines) > termHeight {
		bgLines = bgLines[:termHeight]
	}

	ovLines := strings.Split(overlay, "\n")
	ovHeight := len(ovLines)

	// Find the widest overlay line (visible columns)
	ovWidth := 0
	for _, l := range ovLines {
		if w := width.VisibleWidth(l); w > ovWidth {
			ovWidth = w
		}
	}

	// Center vertically
	startRow := (termHeight - ovHeight) / 2
	if startRow < 0 {
		startRow = 0
	}

	// Center horizontally
	startCol := (termWidth - ovWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Splice overlay lines into background
	for i, ovLine := range ovLines {
		row := startRow + i
		if row >= termHeight {
			break
		}

		bgLine := bgLines[row]
		// Pad background line to startCol visible columns
		bgVis := width.VisibleWidth(bgLine)
		if bgVis < startCol {
			bgLine += strings.Repeat(" ", startCol-bgVis)
		}

		// Build new line: background prefix + overlay + background suffix
		// For simplicity, we replace the overlay region entirely
		prefix := truncateVisual(bgLine, startCol)
		ovVis := width.VisibleWidth(ovLine)

		// Background suffix after the overlay region
		suffix := ""
		afterCol := startCol + ovVis
		if afterCol < termWidth {
			bgAfter := sliceFromCol(bgLines[row], afterCol)
			if bgAfter != "" {
				suffix = bgAfter
			}
		}

		bgLines[row] = prefix + ovLine + suffix
	}

	return strings.Join(bgLines, "\n")
}

// truncateVisual returns the prefix of s that occupies at most maxCols visible columns.
func truncateVisual(s string, maxCols int) string {
	if maxCols <= 0 {
		return ""
	}
	return width.TruncateToWidth(s, maxCols)
}

// sliceFromCol returns the portion of s starting at visible column startCol.
// ANSI sequences are skipped in column counting but preserved in output.
func sliceFromCol(s string, startCol int) string {
	col := 0
	inEsc := false
	for i, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		if col >= startCol {
			return s[i:]
		}
		col++
	}
	return ""
}
