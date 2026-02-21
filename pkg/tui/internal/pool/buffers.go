// ABOUTME: sync.Pool wrappers for bytes.Buffer and strings.Builder
// ABOUTME: Reduces GC pressure for frequent small allocations in rendering

package pool

import (
	"bytes"
	"strings"
	"sync"
)

var bytesBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// GetBytesBuffer returns a bytes.Buffer from the pool.
func GetBytesBuffer() *bytes.Buffer {
	buf := bytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

const maxPoolBufSize = 1024 * 1024 // 1MB: discard oversized buffers

// PutBytesBuffer returns a bytes.Buffer to the pool.
// Buffers exceeding 1MB are discarded to prevent pool bloat.
func PutBytesBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	if buf.Cap() > maxPoolBufSize {
		return // Let GC collect oversized buffers
	}
	buf.Reset()
	bytesBufferPool.Put(buf)
}

var stringBuilderPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

// GetStringBuilder returns a strings.Builder from the pool.
func GetStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// PutStringBuilder returns a strings.Builder to the pool.
// Builders exceeding 1MB are discarded to prevent pool bloat.
func PutStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	if sb.Cap() > maxPoolBufSize {
		return // Let GC collect oversized builders
	}
	sb.Reset()
	stringBuilderPool.Put(sb)
}
