// ABOUTME: Channel-based event streaming for LLM responses
// ABOUTME: EventStream[T] provides type-safe async iteration over SSE events

package ai

import (
	"sync"
	"sync/atomic"
)

// StreamEventType identifies the kind of stream event.
type StreamEventType int

const (
	EventContentDelta StreamEventType = iota
	EventContentDone
	EventToolUseStart
	EventToolUseDelta
	EventToolUseDone
	EventThinkingDelta
	EventMessageStart
	EventMessageDelta
	EventMessageDone
	EventPing
	EventError
)

// StreamEvent represents a single event from the LLM stream.
type StreamEvent struct {
	Type       StreamEventType
	Text       string // Content text delta
	ToolID     string // Tool use ID
	ToolName   string // Tool name
	ToolInput  string // Partial JSON input
	Usage      *Usage
	StopReason StopReason
	Error      error
}

// EventStream provides channel-based access to streaming LLM events.
// Consumers range over Events() and check Result() when done.
type EventStream struct {
	events chan StreamEvent
	done   chan struct{}
	result atomic.Pointer[AssistantMessage]
	once   sync.Once
}

// NewEventStream creates a new EventStream with the given buffer size.
func NewEventStream(bufSize int) *EventStream {
	return &EventStream{
		events: make(chan StreamEvent, bufSize),
		done:   make(chan struct{}),
	}
}

// Events returns a read-only channel of stream events.
// The channel is closed when the stream is complete.
func (s *EventStream) Events() <-chan StreamEvent {
	return s.events
}

// Send sends an event to the stream. Returns false if the stream is closed.
func (s *EventStream) Send(event StreamEvent) bool {
	select {
	case s.events <- event:
		return true
	case <-s.done:
		return false
	}
}

// Finish completes the stream with a final result.
func (s *EventStream) Finish(msg *AssistantMessage) {
	s.once.Do(func() {
		if msg != nil {
			s.result.Store(msg)
		}
		close(s.events)
		close(s.done)
	})
}

// FinishWithError completes the stream with an error event.
func (s *EventStream) FinishWithError(err error) {
	s.Send(StreamEvent{Type: EventError, Error: err})
	s.Finish(nil)
}

// Result blocks until the stream is complete and returns the final message.
func (s *EventStream) Result() *AssistantMessage {
	<-s.done
	return s.result.Load()
}

// Done returns a channel that is closed when the stream completes.
func (s *EventStream) Done() <-chan struct{} {
	return s.done
}
