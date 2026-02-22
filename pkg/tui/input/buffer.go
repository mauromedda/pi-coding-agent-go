// ABOUTME: StdinBuffer reads raw bytes from an io.Reader and dispatches parsed key events.
// ABOUTME: Handles escape sequence buffering, lone-ESC timeout (~50ms), and bracketed paste detection.

package input

import (
	"context"
	"io"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

const (
	readBufSize  = 256
	escTimeout   = 50 * time.Millisecond
	bracketStart = "\x1b[200~"
	bracketEnd   = "\x1b[201~"
)

// StdinBuffer reads from a reader and dispatches parsed key events via onKey.
type StdinBuffer struct {
	reader io.Reader
	onKey  func(key.Key)
	buf    []byte
	mu     sync.Mutex
}

// NewStdinBuffer creates a StdinBuffer that reads from r and calls onKey for each parsed key.
func NewStdinBuffer(r io.Reader, onKey func(key.Key)) *StdinBuffer {
	return &StdinBuffer{
		reader: r,
		onKey:  onKey,
		buf:    make([]byte, 0, readBufSize),
	}
}

// Start reads from the underlying reader until ctx is cancelled or the reader returns an error.
// It blocks until completion; call it in a goroutine if non-blocking behavior is needed.
func (b *StdinBuffer) Start(ctx context.Context) {
	readCh := make(chan readResult)
	done := make(chan struct{})

	go b.readLoop(readCh, done)
	defer close(done)

	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-readCh:
			if !ok {
				b.flushRemaining()
				return
			}
			if result.err != nil {
				b.flushRemaining()
				return
			}
			b.processBytes(ctx, result.data)
		}
	}
}

// readResult holds the outcome of a single Read call.
type readResult struct {
	data []byte
	err  error
}

// readLoop continuously reads from the reader and sends data on ch.
// It stops when done is closed, preventing goroutine leaks on context cancellation.
func (b *StdinBuffer) readLoop(ch chan<- readResult, done <-chan struct{}) {
	defer close(ch)
	tmp := make([]byte, readBufSize)
	for {
		n, err := b.reader.Read(tmp)
		if n > 0 {
			data := make([]byte, n)
			copy(data, tmp[:n])
			select {
			case ch <- readResult{data: data}:
			case <-done:
				return
			}
		}
		if err != nil {
			if n == 0 {
				select {
				case ch <- readResult{err: err}:
				case <-done:
				}
			}
			return
		}
	}
}

// processBytes appends incoming data to the internal buffer and dispatches complete keys.
func (b *StdinBuffer) processBytes(ctx context.Context, data []byte) {
	b.mu.Lock()
	b.buf = append(b.buf, data...)
	b.mu.Unlock()

	b.dispatchKeys(ctx)
}

// dispatchKeys parses and dispatches all complete key sequences from the buffer.
func (b *StdinBuffer) dispatchKeys(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		b.mu.Lock()
		if len(b.buf) == 0 {
			b.mu.Unlock()
			return
		}

		consumed, k, needsWait := b.tryParse()
		if needsWait {
			b.mu.Unlock()
			// Lone ESC: wait briefly for more bytes, then re-check.
			if !b.waitForMore(ctx) {
				return
			}
			continue
		}

		if consumed > 0 {
			b.buf = b.buf[consumed:]
		}
		b.mu.Unlock()

		if consumed > 0 {
			b.onKey(k)
		} else {
			return
		}
	}
}

// tryParse attempts to parse one key from the front of b.buf.
// Returns (consumed bytes, parsed key, needs-wait flag).
// Must be called with b.mu held.
func (b *StdinBuffer) tryParse() (int, key.Key, bool) {
	if len(b.buf) == 0 {
		return 0, key.Key{}, false
	}

	// Check for bracketed paste; skip the paste content entirely.
	if consumed := b.skipBracketedPaste(); consumed > 0 {
		return consumed, key.Key{Type: key.KeyUnknown}, false
	}

	// Escape sequence: need at least 2 bytes to distinguish ESC from escape seq.
	if b.buf[0] == 0x1b {
		if len(b.buf) == 1 {
			// Might be lone ESC or start of sequence; caller should wait.
			return 0, key.Key{}, true
		}
		return b.parseEscapeFromBuf()
	}

	// Check for incomplete UTF-8 rune; wait for more bytes if the buffer
	// is shorter than the maximum rune length.
	if !utf8.FullRune(b.buf) {
		if len(b.buf) < utf8.UTFMax {
			return 0, key.Key{}, true
		}
		// Buffer is long enough but still invalid; consume one byte.
		return 1, key.Key{Type: key.KeyUnknown}, false
	}

	r, size := utf8.DecodeRune(b.buf)
	if r == utf8.RuneError {
		return 1, key.Key{Type: key.KeyUnknown}, false
	}

	k := key.ParseKey(string(b.buf[:size]))
	return size, k, false
}

// parseEscapeFromBuf parses an escape sequence from the buffer.
// Must be called with b.mu held and len(b.buf) >= 2.
func (b *StdinBuffer) parseEscapeFromBuf() (int, key.Key, bool) {
	// Try progressively longer prefixes (max 6 bytes for sequences like \x1b[200~).
	maxLen := min(len(b.buf), 8)

	// Try longest match first, then shorter.
	for end := maxLen; end >= 2; end-- {
		candidate := string(b.buf[:end])
		k := key.ParseKey(candidate)
		if k.Type != key.KeyUnknown {
			return end, k, false
		}
	}

	// No known sequence matched; could be incomplete or truly unknown.
	// If buffer is short and second byte indicates CSI/SS3, wait for more.
	if len(b.buf) <= 3 && (b.buf[1] == '[' || b.buf[1] == 'O') {
		return 0, key.Key{}, true
	}

	// Unknown sequence; consume the ESC and let the rest be re-parsed.
	return 1, key.Key{Type: key.KeyEscape}, false
}

// skipBracketedPaste detects and skips bracketed paste content.
// Returns the number of bytes consumed (0 if no bracketed paste found).
// Must be called with b.mu held.
func (b *StdinBuffer) skipBracketedPaste() int {
	s := string(b.buf)
	if len(s) < len(bracketStart) {
		return 0
	}
	if s[:len(bracketStart)] != bracketStart {
		return 0
	}
	// Find the end marker
	for i := len(bracketStart); i <= len(s)-len(bracketEnd); i++ {
		if s[i:i+len(bracketEnd)] == bracketEnd {
			return i + len(bracketEnd)
		}
	}
	// End marker not found yet; wait for more data.
	return 0
}

// waitForMore pauses briefly to allow more bytes to arrive for escape sequence completion.
// Returns false if context was cancelled.
func (b *StdinBuffer) waitForMore(ctx context.Context) bool {
	timer := time.NewTimer(escTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		// Timeout: treat buffered ESC as lone Escape.
		b.mu.Lock()
		if len(b.buf) > 0 && b.buf[0] == 0x1b && len(b.buf) == 1 {
			b.buf = b.buf[1:]
			b.mu.Unlock()
			b.onKey(key.Key{Type: key.KeyEscape})
			return true
		}
		b.mu.Unlock()
		return true
	}
}

// flushRemaining dispatches any leftover bytes in the buffer.
func (b *StdinBuffer) flushRemaining() {
	b.mu.Lock()
	for len(b.buf) > 0 {
		consumed, k, needsWait := b.tryParse()
		if needsWait {
			// No more data coming; treat as lone escape.
			b.buf = b.buf[1:]
			b.mu.Unlock()
			b.onKey(key.Key{Type: key.KeyEscape})
			b.mu.Lock()
			continue
		}
		if consumed > 0 {
			b.buf = b.buf[consumed:]
			b.mu.Unlock()
			b.onKey(k)
			b.mu.Lock()
		} else {
			break
		}
	}
	b.mu.Unlock()
}
