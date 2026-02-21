// ABOUTME: SGR state machine that tracks active text styling
// ABOUTME: Processes CSI SGR sequences and emits minimal restore sequences

package ansitrack

import (
	"strconv"
	"strings"
)

// Tracker maintains the current SGR (Select Graphic Rendition) state.
type Tracker struct {
	bold       bool
	dim        bool
	italic     bool
	underline  bool
	blink      bool
	reverse    bool
	hidden     bool
	strikethrough bool
	fg         string // e.g., "31" or "38;5;196"
	bg         string
}

// Reset clears all SGR state.
func (t *Tracker) Reset() {
	*t = Tracker{}
}

// Process applies an SGR escape sequence to the tracker state.
// The sequence should be a complete CSI SGR (e.g., "\x1b[31;1m").
func (t *Tracker) Process(seq string) {
	if !strings.HasPrefix(seq, "\x1b[") || !strings.HasSuffix(seq, "m") {
		return
	}
	params := seq[2 : len(seq)-1]
	if params == "" || params == "0" {
		t.Reset()
		return
	}

	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		code, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}
		switch {
		case code == 0:
			t.Reset()
		case code == 1:
			t.bold = true
		case code == 2:
			t.dim = true
		case code == 3:
			t.italic = true
		case code == 4:
			t.underline = true
		case code == 5:
			t.blink = true
		case code == 7:
			t.reverse = true
		case code == 8:
			t.hidden = true
		case code == 9:
			t.strikethrough = true
		case code >= 30 && code <= 37:
			t.fg = parts[i]
		case code == 38:
			// Extended foreground: 38;5;N or 38;2;R;G;B
			t.fg = strings.Join(parts[i:], ";")
			return // Consumed remaining
		case code >= 40 && code <= 47:
			t.bg = parts[i]
		case code == 48:
			t.bg = strings.Join(parts[i:], ";")
			return
		case code == 39:
			t.fg = ""
		case code == 49:
			t.bg = ""
		}
	}
}

// Restore returns the minimal SGR sequence to re-establish current state.
// Returns empty string if no styling is active.
func (t *Tracker) Restore() string {
	var codes []string

	if t.bold {
		codes = append(codes, "1")
	}
	if t.dim {
		codes = append(codes, "2")
	}
	if t.italic {
		codes = append(codes, "3")
	}
	if t.underline {
		codes = append(codes, "4")
	}
	if t.blink {
		codes = append(codes, "5")
	}
	if t.reverse {
		codes = append(codes, "7")
	}
	if t.hidden {
		codes = append(codes, "8")
	}
	if t.strikethrough {
		codes = append(codes, "9")
	}
	if t.fg != "" {
		codes = append(codes, t.fg)
	}
	if t.bg != "" {
		codes = append(codes, t.bg)
	}

	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// IsActive returns true if any SGR state is set.
func (t *Tracker) IsActive() bool {
	return t.bold || t.dim || t.italic || t.underline || t.blink ||
		t.reverse || t.hidden || t.strikethrough || t.fg != "" || t.bg != ""
}
