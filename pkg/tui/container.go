// ABOUTME: Container is an ordered collection of child Components
// ABOUTME: Thread-safe via RWMutex for concurrent render vs mutation

package tui

import "sync"

// Container holds an ordered list of child components.
// It is safe for concurrent access: mutations acquire a write lock,
// rendering acquires a read lock.
type Container struct {
	mu       sync.RWMutex
	children []Component
}

// NewContainer creates an empty Container.
func NewContainer() *Container {
	return &Container{}
}

// Add appends a component to the container.
func (c *Container) Add(comp Component) {
	c.mu.Lock()
	c.children = append(c.children, comp)
	c.mu.Unlock()
}

// Remove removes a component from the container.
// Returns true if the component was found and removed.
func (c *Container) Remove(comp Component) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, child := range c.children {
		if child == comp {
			c.children = append(c.children[:i], c.children[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all children.
func (c *Container) Clear() {
	c.mu.Lock()
	c.children = c.children[:0]
	c.mu.Unlock()
}

// Children returns a snapshot of the current children.
func (c *Container) Children() []Component {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Component, len(c.children))
	copy(out, c.children)
	return out
}

// Render renders all children sequentially into the buffer.
func (c *Container) Render(out *RenderBuffer, width int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, child := range c.children {
		child.Render(out, width)
	}
}

// Invalidate invalidates all children.
func (c *Container) Invalidate() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, child := range c.children {
		child.Invalidate()
	}
}
