// ABOUTME: Server-Sent Events parser that reads from an io.Reader
// ABOUTME: Supports event, data, id fields; multi-line data; comments

package sse

import (
	"bufio"
	"io"
	"strings"
)

// Event represents a single Server-Sent Event.
type Event struct {
	Type string
	Data string
	ID   string
}

// Reader parses Server-Sent Events from an io.Reader.
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader creates a new SSE reader from the given io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// Next reads and returns the next SSE event.
// Returns nil, io.EOF when the stream ends.
func (r *Reader) Next() (*Event, error) {
	var ev Event
	var hasContent bool

	for r.scanner.Scan() {
		line := r.scanner.Text()

		if line == "" {
			if hasContent {
				return &ev, nil
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value := parseLine(line)
		hasContent = applyField(&ev, field, value, hasContent)
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	if hasContent {
		return &ev, nil
	}

	return nil, io.EOF
}

// parseLine splits an SSE line into field name and value.
func parseLine(line string) (string, string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return line, ""
	}

	field := line[:idx]
	value := line[idx+1:]

	// Strip optional leading space after colon.
	if len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}

	return field, value
}

// applyField applies a parsed field to the event and returns whether the event has content.
func applyField(ev *Event, field, value string, hadContent bool) bool {
	switch field {
	case "event":
		ev.Type = value
		return true
	case "data":
		if ev.Data != "" {
			ev.Data += "\n"
		}
		ev.Data += value
		return true
	case "id":
		ev.ID = value
		return true
	default:
		return hadContent
	}
}
