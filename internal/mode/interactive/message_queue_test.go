// ABOUTME: Tests for message queue component

package interactive

import (
	"testing"
)

func TestMessageQueue_Create(t *testing.T) {
	mq := NewMessageQueue()
	if mq == nil {
		t.Fatal("NewMessageQueue returned nil")
	}
	if mq.Count() != 0 {
		t.Errorf("Expected 0 messages, got %d", mq.Count())
	}
}

func TestMessageQueue_PushPop(t *testing.T) {
	mq := NewMessageQueue()

	mq.Push("message1")
	mq.Push("message2")

	if mq.Count() != 2 {
		t.Errorf("Expected 2 messages, got %d", mq.Count())
	}

	msg1 := mq.Pop()
	if msg1 != "message1" {
		t.Errorf("Expected 'message1', got '%s'", msg1)
	}

	if mq.Count() != 1 {
		t.Errorf("Expected 1 message after pop, got %d", mq.Count())
	}

	msg2 := mq.Pop()
	if msg2 != "message2" {
		t.Errorf("Expected 'message2', got '%s'", msg2)
	}

	if mq.Count() != 0 {
		t.Errorf("Expected 0 messages after all pops, got %d", mq.Count())
	}
}

func TestMessageQueue_PopEmpty(t *testing.T) {
	mq := NewMessageQueue()
	msg := mq.Pop()

	if msg != "" {
		t.Errorf("Expected empty string for empty queue, got '%s'", msg)
	}
}

func TestMessageQueue_Current(t *testing.T) {
	mq := NewMessageQueue()

	msg := mq.Current()
	if msg != "" {
		t.Errorf("Expected empty string for no current message, got '%s'", msg)
	}

	mq.Push("test")
	msg = mq.Current()
	if msg != "test" {
		t.Errorf("Expected 'test' as current, got '%s'", msg)
	}
}

func TestMessageQueue_HasMessages(t *testing.T) {
	mq := NewMessageQueue()

	if mq.HasMessages() {
		t.Error("Expected no messages initially")
	}

	mq.Push("test")

	if !mq.HasMessages() {
		t.Error("Expected messages after push")
	}
}

func TestMessageQueue_Clear(t *testing.T) {
	mq := NewMessageQueue()
	mq.Push("a")
	mq.Push("b")

	mq.Clear()

	if mq.Count() != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", mq.Count())
	}
}

func TestMessageQueue_Replace(t *testing.T) {
	mq := NewMessageQueue()
	mq.Push("original")

	mq.Replace("updated")

	if mq.Current() != "updated" {
		t.Errorf("Expected 'updated', got '%s'", mq.Current())
	}
}

func TestMessageQueue_NextPrev(t *testing.T) {
	mq := NewMessageQueue()
	mq.Push("first")
	mq.Push("second")
	mq.Push("third")

	// Start from beginning
	msg := mq.Next()
	if msg != "first" {
		t.Errorf("Expected 'first', got '%s'", msg)
	}

	msg = mq.Next()
	if msg != "second" {
		t.Errorf("Expected 'second', got '%s'", msg)
	}

	msg = mq.Next()
	if msg != "third" {
		t.Errorf("Expected 'third', got '%s'", msg)
	}

	msg = mq.Prev()
	if msg != "second" {
		t.Errorf("Expected 'second', got '%s'", msg)
	}

	msg = mq.Prev()
	if msg != "first" {
		t.Errorf("Expected 'first', got '%s'", msg)
	}
}

func TestMessageQueue_NextAtEnd(t *testing.T) {
	mq := NewMessageQueue()
	mq.Push("only")

	msg := mq.Next()
	if msg != "only" {
		t.Errorf("Expected 'only', got '%s'", msg)
	}

	msg = mq.Next()
	if msg != "only" {
		t.Errorf("Expected 'only' at end (stays at last), got '%s'", msg)
	}
}

func TestMessageQueue_Retrieve(t *testing.T) {
	mq := NewMessageQueue()
	mq.Push("a")
	mq.Push("b")
	mq.Push("c")

	msg := mq.Retrieve(1)
	if msg != "b" {
		t.Errorf("Expected 'b', got '%s'", msg)
	}

	msg = mq.Retrieve(10)
	if msg != "" {
		t.Errorf("Expected empty string for out of range, got '%s'", msg)
	}
}
