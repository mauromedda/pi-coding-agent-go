// ABOUTME: Emacs-style kill ring buffer for cut/yank operations
// ABOUTME: Fixed-size circular buffer; supports yank and yank-pop

package killring

const defaultSize = 32

// KillRing is an Emacs-style ring buffer for killed (cut) text.
type KillRing struct {
	entries []string
	pos     int
	size    int
	yankIdx int
}

// New creates a KillRing with the default capacity.
func New() *KillRing {
	return &KillRing{
		entries: make([]string, 0, defaultSize),
		size:    defaultSize,
	}
}

// Push adds text to the kill ring.
func (kr *KillRing) Push(text string) {
	if len(kr.entries) < kr.size {
		kr.entries = append(kr.entries, text)
	} else {
		kr.entries[kr.pos] = text
	}
	kr.pos = (kr.pos + 1) % kr.size
	kr.yankIdx = kr.pos
}

// Yank returns the most recently killed text, or empty if ring is empty.
func (kr *KillRing) Yank() string {
	if len(kr.entries) == 0 {
		return ""
	}
	idx := (kr.pos - 1 + len(kr.entries)) % len(kr.entries)
	kr.yankIdx = idx
	return kr.entries[idx]
}

// YankPop cycles to the next older entry in the ring.
// Should only be called after Yank.
func (kr *KillRing) YankPop() string {
	if len(kr.entries) == 0 {
		return ""
	}
	kr.yankIdx = (kr.yankIdx - 1 + len(kr.entries)) % len(kr.entries)
	return kr.entries[kr.yankIdx]
}

// Len returns the number of entries in the ring.
func (kr *KillRing) Len() int {
	return len(kr.entries)
}
