// ABOUTME: iTerm2 inline images protocol encoder
// ABOUTME: Generates OSC 1337 escape sequences for inline image display

package image

import (
	"encoding/base64"
	"fmt"
)

// EncodeITerm2 encodes image data into an iTerm2 inline image escape sequence.
// The width parameter controls display width (e.g. "auto", "80", "50%").
func EncodeITerm2(data []byte, width string) string {
	if len(data) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("\x1b]1337;File=inline=1;size=%d;width=%s:%s\a", len(data), width, encoded)
}
