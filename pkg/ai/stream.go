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
//
// Design: Send writes to an internal events channel that is never closed
// externally. Finish closes only the done channel. A drainer goroutine
// forwards events to the consumer-facing out channel, closing it when
// done fires and all buffered events are drained. This eliminates the
// send-on-closed-channel race between Send and Finish.
type EventStream struct {
	events chan StreamEvent // internal: producers write here via Send
	out    chan StreamEvent // external: consumers read via Events()
	done   chan struct{}
	result atomic.Pointer[AssistantMessage]
	once   sync.Once
}

// NewEventStream creates a new EventStream with the given buffer size.
func NewEventStream(bufSize int) *EventStream {
	s := &EventStream{
		events: make(chan StreamEvent, bufSize),
		out:    make(chan StreamEvent, bufSize),
		done:   make(chan struct{}),
	}
	go s.drain()
	return s
}

// drain forwards events from the internal channel to the consumer channel.
// Closes out when done fires and all buffered events are forwarded.
func (s *EventStream) drain() {
	defer close(s.out)
	for {
		select {
		case ev := <-s.events:
			s.out <- ev
		case <-s.done:
			// Drain remaining buffered events.
			for {
				select {
				case ev := <-s.events:
					s.out <- ev
				default:
					return
				}
			}
		}
	}
}

// Events returns a read-only channel of stream events.
// The channel is closed when the stream is complete.
func (s *EventStream) Events() <-chan StreamEvent {
	return s.out
}

// Send sends an event to the stream. Returns false if the stream is finished.
func (s *EventStream) Send(event StreamEvent) bool {
	select {
	case <-s.done:
		return false
	default:
	}
	select {
	case s.events <- event:
		return true
	case <-s.done:
		return false
	}
}

// Finish completes the stream with a final result.
// Only closes the done channel; the events channel is never closed, preventing
// send-on-closed-channel panics. The drainer goroutine closes the consumer
// channel after draining remaining events.
func (s *EventStream) Finish(msg *AssistantMessage) {
	s.once.Do(func() {
		if msg != nil {
			s.result.Store(msg)
		}
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
