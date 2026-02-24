// ABOUTME: Tests for EventStream send/receive, finish, and done channel behavior
// ABOUTME: Validates channel-based streaming lifecycle and result retrieval

package ai

import (
	"errors"
	"testing"
	"time"
)

func TestEventStreamSendAndReceive(t *testing.T) {
	t.Parallel()

	stream := NewEventStream(10)

	sent := StreamEvent{Type: EventContentDelta, Text: "hello"}
	ok := stream.Send(sent)
	if !ok {
		t.Fatal("Send returned false; expected true")
	}

	select {
	case got := <-stream.Events():
		if got.Type != sent.Type {
			t.Errorf("got Type %v, want %v", got.Type, sent.Type)
		}
		if got.Text != sent.Text {
			t.Errorf("got Text %q, want %q", got.Text, sent.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventStreamFinishWithResult(t *testing.T) {
	t.Parallel()

	stream := NewEventStream(10)

	msg := &AssistantMessage{
		Content:    []Content{{Type: ContentText, Text: "response"}},
		StopReason: StopEndTurn,
		Usage:      Usage{InputTokens: 10, OutputTokens: 5},
		Model:      "test-model",
	}

	stream.Finish(msg)

	result := stream.Result()
	if result == nil {
		t.Fatal("Result() returned nil")
	}
	if result.Model != "test-model" {
		t.Errorf("got Model %q, want %q", result.Model, "test-model")
	}
	if result.StopReason != StopEndTurn {
		t.Errorf("got StopReason %v, want %v", result.StopReason, StopEndTurn)
	}

	// Events channel should be closed.
	_, open := <-stream.Events()
	if open {
		t.Error("Events channel still open after Finish")
	}
}

func TestEventStreamFinishWithError(t *testing.T) {
	t.Parallel()

	stream := NewEventStream(10)
	testErr := errors.New("test error")

	stream.FinishWithError(testErr)

	// Should receive the error event.
	var gotError bool
	for ev := range stream.Events() {
		if ev.Type == EventError && ev.Error != nil {
			if ev.Error.Error() != testErr.Error() {
				t.Errorf("got error %q, want %q", ev.Error, testErr)
			}
			gotError = true
		}
	}
	if !gotError {
		t.Error("did not receive error event")
	}

	// Result should be nil after FinishWithError.
	result := stream.Result()
	if result != nil {
		t.Errorf("Result() = %v, want nil", result)
	}
}

func TestEventStreamDoneChannel(t *testing.T) {
	t.Parallel()

	stream := NewEventStream(10)

	// Done channel should not be closed yet.
	select {
	case <-stream.Done():
		t.Fatal("Done() closed before Finish")
	default:
		// expected
	}

	stream.Finish(nil)

	// Done channel should now be closed.
	select {
	case <-stream.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("Done() not closed after Finish")
	}
}

func TestEventStreamConcurrentSendAndFinish(t *testing.T) {
	t.Parallel()

	// This test verifies that concurrent Send and Finish calls do not panic.
	// The race was: Finish closes events channel while Send writes to it.
	for i := range 100 {
		_ = i
		stream := NewEventStream(1)

		done := make(chan struct{})
		go func() {
			defer close(done)
			for j := range 10 {
				_ = j
				stream.Send(StreamEvent{Type: EventContentDelta, Text: "delta"})
			}
		}()

		// Finish concurrently; must not panic.
		stream.Finish(nil)
		<-done
	}
}

func TestEventStreamSendReturnsFalseAfterFinish(t *testing.T) {
	t.Parallel()

	stream := NewEventStream(1)
	stream.Finish(nil)

	// After Finish, Send must return false without panic.
	ok := stream.Send(StreamEvent{Type: EventContentDelta, Text: "late"})
	if ok {
		t.Error("Send returned true after Finish; want false")
	}
}

func TestEventStreamDoubleFinish(t *testing.T) {
	t.Parallel()

	stream := NewEventStream(10)
	msg := &AssistantMessage{Model: "first"}

	// Double finish should not panic (sync.Once guarantees this).
	stream.Finish(msg)
	stream.Finish(&AssistantMessage{Model: "second"})

	result := stream.Result()
	if result == nil || result.Model != "first" {
		t.Errorf("expected first finish result, got %v", result)
	}
}
