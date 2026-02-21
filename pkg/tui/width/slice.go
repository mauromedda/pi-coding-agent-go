// ABOUTME: Column-based string slicing with ANSI-awareness
// ABOUTME: SliceByColumn extracts a visual range from styled text

package width

import "github.com/rivo/uniseg"

// SliceByColumn extracts the substring from column start (inclusive) to
// column end (exclusive), preserving ANSI escape sequences.
// Columns are zero-indexed visual positions.
func SliceByColumn(s string, start, end int) string {
	if start >= end || s == "" {
		return ""
	}

	type segment struct {
		text  string
		col   int
		width int
		isSeq bool
	}

	segments := extractSegments(s)
	var result []byte
	for _, seg := range segments {
		if seg.isSeq {
			// Always include ANSI sequences that fall within or before our range
			result = append(result, seg.text...)
			continue
		}
		segEnd := seg.col + seg.width
		if segEnd <= start || seg.col >= end {
			continue
		}
		// This segment overlaps with [start, end)
		result = append(result, seg.text...)
	}
	return string(result)
}

// segment represents either a visible grapheme cluster or an ANSI sequence.
type segment struct {
	text  string
	col   int
	width int
	isSeq bool
}

// extractSegments breaks a string into segments of visible text and ANSI sequences.
func extractSegments(s string) []segment {
	var segs []segment
	col := 0
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			end := skipANSISequence(s, i)
			segs = append(segs, segment{text: s[i:end], col: col, isSeq: true})
			i = end
			continue
		}
		// Read one grapheme cluster
		cluster, rest, _, _ := uniseg.FirstGraphemeClusterInString(s[i:], -1)
		w := graphemeWidth(cluster)
		segs = append(segs, segment{text: cluster, col: col, width: w})
		col += w
		i += len(s[i:]) - len(rest)
	}
	return segs
}
