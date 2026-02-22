// ABOUTME: Message queue for follow-up and steer messages
// ABOUTME: Supports Alt+Enter for follow-up, Enter for steer, Alt+Up/Down for navigation
// ABOUTME: Updated to support message editing while queued

package interactive

import (
	"sync"
)

// MessageQueue manages queued messages for follow-up and steer behaviors
type MessageQueue struct {
	mu        sync.Mutex
	messages  []string // Queued messages
	queuedIdx int      // Index of currently queued message
	editIdx   int      // Index of message being edited (-1 if none)
}

// NewMessageQueue creates a new message queue
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages:  make([]string, 0),
		queuedIdx: -1,
		editIdx:   -1,
	}
}

// Push adds a message to the queue and sets it as current
func (mq *MessageQueue) Push(msg string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	mq.messages = append(mq.messages, msg)
	mq.queuedIdx = len(mq.messages) - 1
	mq.editIdx = -1 // Clear edit mode when pushing new message
}

// Pop removes and returns the oldest message from the queue
func (mq *MessageQueue) Pop() string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) == 0 {
		return ""
	}

	msg := mq.messages[0]
	mq.messages = mq.messages[1:]
	if mq.queuedIdx >= 0 {
		mq.queuedIdx--
		if mq.queuedIdx < 0 {
			mq.queuedIdx = -1
		}
	}
	if mq.editIdx >= 0 {
		mq.editIdx--
		if mq.editIdx < 0 {
			mq.editIdx = -1
		}
	}
	return msg
}

// Current returns the currently queued message (for follow-up)
func (mq *MessageQueue) Current() string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.queuedIdx < 0 || mq.queuedIdx >= len(mq.messages) {
		return ""
	}
	return mq.messages[mq.queuedIdx]
}

// Count returns the number of queued messages
func (mq *MessageQueue) Count() int {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.messages)
}

// HasMessages returns true if there are queued messages
func (mq *MessageQueue) HasMessages() bool {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.messages) > 0
}

// Clear removes all messages from the queue
func (mq *MessageQueue) Clear() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	mq.messages = make([]string, 0)
	mq.queuedIdx = -1
	mq.editIdx = -1
}

// Replace replaces the current message with a new one
func (mq *MessageQueue) Replace(msg string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.queuedIdx < 0 || mq.queuedIdx >= len(mq.messages) {
		mq.messages = append(mq.messages, msg)
		mq.queuedIdx = len(mq.messages) - 1
		mq.editIdx = -1
	} else {
		mq.messages[mq.queuedIdx] = msg
	}
}

// EditMode returns true if edit mode is active
func (mq *MessageQueue) EditMode() bool {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return mq.editIdx >= 0
}

// StartEdit starts editing the current message
func (mq *MessageQueue) StartEdit() string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.queuedIdx < 0 || mq.queuedIdx >= len(mq.messages) {
		return ""
	}
	mq.editIdx = mq.queuedIdx
	return mq.messages[mq.queuedIdx]
}

// EditMessage updates the message being edited
func (mq *MessageQueue) EditMessage(msg string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.editIdx < 0 || mq.editIdx >= len(mq.messages) {
		return
	}
	mq.messages[mq.editIdx] = msg
}

// CommitEdit commits the edit and returns to normal mode
func (mq *MessageQueue) CommitEdit() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.editIdx = -1
}

// CancelEdit cancels the edit and returns to normal mode
func (mq *MessageQueue) CancelEdit() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.editIdx = -1
}

// Retrieve retrieves a message from a specific index
func (mq *MessageQueue) Retrieve(idx int) string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if idx < 0 || idx >= len(mq.messages) {
		return ""
	}
	return mq.messages[idx]
}

// Next retrieves the next message (cycles to first after last)
func (mq *MessageQueue) Next() string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) == 0 {
		return ""
	}

	if mq.queuedIdx < 0 {
		// First call - start from beginning
		mq.queuedIdx = 0
		return mq.messages[0]
	}

	// Increment and cycle
	mq.queuedIdx++
	if mq.queuedIdx >= len(mq.messages) {
		mq.queuedIdx = 0
	}
	return mq.messages[mq.queuedIdx]
}

// Prev retrieves the previous message before the current one
func (mq *MessageQueue) Prev() string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.queuedIdx <= 0 || len(mq.messages) == 0 {
		return ""
	}

	mq.queuedIdx--
	return mq.messages[mq.queuedIdx]
}

// Messages returns a copy of all queued messages
func (mq *MessageQueue) Messages() []string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	msgs := make([]string, len(mq.messages))
	copy(msgs, mq.messages)
	return msgs
}
