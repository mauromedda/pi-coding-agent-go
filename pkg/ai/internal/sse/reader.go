// ABOUTME: Server-Sent Events parser that reads from an io.Reader
// ABOUTME: Supports event, data, id fields; multi-line data; comments

package sse

import (
	"bufio"
	"io"
	"strings"
	"sync"
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
	buf     []byte // pooled buffer, returned via Close()
}

const maxLineSize = 1024 * 1024 // 1MB max line size
const initialBufSize = 64 * 1024 // 64KB initial scanner buffer

// bufPool reuses 64KB scanner buffers across SSE streams.
var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, initialBufSize)
	},
}

// NewReader creates a new SSE reader from the given io.Reader.
// Call Close() when done to return the buffer to the pool.
func NewReader(r io.Reader) *Reader {
	buf := bufPool.Get().([]byte)
	buf = buf[:0]
	s := bufio.NewScanner(r)
	s.Buffer(buf, maxLineSize)
	return &Reader{
		scanner: s,
		buf:     buf,
	}
}

// Close returns the internal buffer to the pool for reuse.
func (r *Reader) Close() {
	if r.buf != nil {
		bufPool.Put(r.buf[:0])
		r.buf = nil
	}
}

// Next reads and returns the next SSE event.
// Returns nil, io.EOF when the stream ends.
func (r *Reader) Next() (*Event, error) {
	var ev Event
	var dataLines []string
	var hasContent bool

	for r.scanner.Scan() {
		line := r.scanner.Text()

		if line == "" {
			if hasContent {
				if len(dataLines) > 0 {
					ev.Data = strings.Join(dataLines, "\n")
				}
				return &ev, nil
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value := parseLine(line)
		hasContent = applyField(&ev, &dataLines, field, value, hasContent)
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	if hasContent {
		if len(dataLines) > 0 {
			ev.Data = strings.Join(dataLines, "\n")
		}
		return &ev, nil
	}

	return nil, io.EOF
}

// parseLine splits an SSE line into field name and value.
func parseLine(line string) (string, string) {
	before, after, ok := strings.Cut(line, ":")
	if !ok {
		return line, ""
	}

	field := before
	value := after

	// Strip optional leading space after colon.
	if len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}

	return field, value
}

// applyField applies a parsed field to the event and returns whether the event has content.
// Data lines are accumulated in dataLines to avoid repeated string concatenation;
// they are joined once when the event is complete.
func applyField(ev *Event, dataLines *[]string, field, value string, hadContent bool) bool {
	switch field {
	case "event":
		ev.Type = value
		return true
	case "data":
		*dataLines = append(*dataLines, value)
		return true
	case "id":
		ev.ID = value
		return true
	default:
		return hadContent
	}
}
