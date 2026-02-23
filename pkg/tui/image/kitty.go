// ABOUTME: Kitty graphics protocol encoder with chunked base64 transmission
// ABOUTME: Implements APC-based image display for Kitty, Ghostty, and WezTerm

package image

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const kittyChunkSize = 4096 // Max base64 chars per chunk

// EncodeKitty encodes PNG data into Kitty graphics protocol escape sequences.
// The output uses chunked transmission: first chunk carries the full header,
// continuation chunks carry only the m= (more) flag and payload.
func EncodeKitty(pngData []byte, cols, rows int) string {
	if len(pngData) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(pngData)

	var b strings.Builder

	if len(encoded) <= kittyChunkSize {
		// Single chunk: m=0 (no more data)
		fmt.Fprintf(&b, "\x1b_Ga=T,f=100,q=2,c=%d,r=%d,m=0;%s\x1b\\", cols, rows, encoded)
		return b.String()
	}

	// Multi-chunk transmission
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
			// First chunk: full header
			fmt.Fprintf(&b, "\x1b_Ga=T,f=100,q=2,c=%d,r=%d,m=%d;%s\x1b\\", cols, rows, more, chunk)
		} else {
			// Continuation chunk
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}

	return b.String()
}
