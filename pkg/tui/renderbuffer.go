// ABOUTME: Pooled line buffer for TUI rendering; recycled via sync.Pool
// ABOUTME: Components write lines here; TUI engine diffs against previous frame

package tui

import "sync"

var bufferPool = sync.Pool{
	New: func() any {
		return &RenderBuffer{
			Lines: make([]string, 0, 64),
		}
	},
}

// AcquireBuffer gets a RenderBuffer from the pool.
func AcquireBuffer() *RenderBuffer {
	buf := bufferPool.Get().(*RenderBuffer)
	buf.Reset()
	return buf
}

// ReleaseBuffer returns a RenderBuffer to the pool.
func ReleaseBuffer(buf *RenderBuffer) {
	if buf == nil {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

// RenderBuffer is a pooled line buffer that components write into.
// The TUI engine allocates from sync.Pool and recycles after each frame.
type RenderBuffer struct {
	Lines []string
}

// WriteLine appends a single line to the buffer.
func (b *RenderBuffer) WriteLine(line string) {
	b.Lines = append(b.Lines, line)
}

// WriteLines appends multiple lines to the buffer.
func (b *RenderBuffer) WriteLines(lines []string) {
	b.Lines = append(b.Lines, lines...)
}

// Reset clears the buffer for reuse without deallocating.
func (b *RenderBuffer) Reset() {
	b.Lines = b.Lines[:0]
}

// Len returns the number of lines in the buffer.
func (b *RenderBuffer) Len() int {
	return len(b.Lines)
}
