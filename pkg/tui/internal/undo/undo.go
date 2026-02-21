// ABOUTME: Generic undo stack for reversible operations
// ABOUTME: Type-parameterized; supports push, undo, and redo

package undo

// Stack is a generic undo/redo stack.
type Stack[S any] struct {
	undoStack []S
	redoStack []S
	maxSize   int
}

// New creates an undo Stack with the given maximum depth.
func New[S any](maxSize int) *Stack[S] {
	return &Stack[S]{
		undoStack: make([]S, 0, maxSize),
		redoStack: make([]S, 0, maxSize),
		maxSize:   maxSize,
	}
}

// Push saves a state snapshot onto the undo stack, clearing redo history.
func (s *Stack[S]) Push(state S) {
	if len(s.undoStack) >= s.maxSize {
		// Evict oldest
		s.undoStack = s.undoStack[1:]
	}
	s.undoStack = append(s.undoStack, state)
	s.redoStack = s.redoStack[:0]
}

// Undo pops the most recent state. Returns the state and true, or zero
// value and false if there is nothing to undo.
func (s *Stack[S]) Undo() (S, bool) {
	if len(s.undoStack) == 0 {
		var zero S
		return zero, false
	}
	last := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.redoStack = append(s.redoStack, last)
	return last, true
}

// Redo re-applies the most recently undone state.
func (s *Stack[S]) Redo() (S, bool) {
	if len(s.redoStack) == 0 {
		var zero S
		return zero, false
	}
	last := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]
	s.undoStack = append(s.undoStack, last)
	return last, true
}

// CanUndo returns true if there are states to undo.
func (s *Stack[S]) CanUndo() bool {
	return len(s.undoStack) > 0
}

// CanRedo returns true if there are states to redo.
func (s *Stack[S]) CanRedo() bool {
	return len(s.redoStack) > 0
}
