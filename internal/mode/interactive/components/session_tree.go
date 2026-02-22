// ABOUTME: Session tree viewer component for /tree command
// ABOUTME: Displays session history as a branch graph with filter support

package components

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/session"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/fuzzy"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

// SessionNode represents a session entry in the tree
type SessionNode struct {
	ID       string
	ParentID string
	Start    *session.SessionStartData
	Model    string
	Count    int
	Children []*SessionNode
	Level    int
	IsBranch bool
}

// SessionTree is a tree viewer for session history
type SessionTree struct {
	roots     []*SessionNode
	filter    string
	selected  int
	scrollOff int
	maxHeight int
	dirty     bool
	mu        sync.Mutex
}

// NewSessionTree creates a new session tree viewer
func NewSessionTree(roots []*SessionNode) *SessionTree {
	return &SessionTree{
		roots:     roots,
		maxHeight: 20,
		dirty:     true,
	}
}

// SetFilter sets the search filter string
func (st *SessionTree) SetFilter(f string) {
	st.mu.Lock()
	st.filter = f
	st.selected = 0
	st.scrollOff = 0
	st.dirty = true
	st.mu.Unlock()
}

// SetMaxHeight sets the maximum visible rows
func (st *SessionTree) SetMaxHeight(h int) {
	st.mu.Lock()
	st.maxHeight = h
	st.dirty = true
	st.mu.Unlock()
}

// SelectedNode returns the currently selected node
func (st *SessionTree) SelectedNode() *SessionNode {
	st.mu.Lock()
	defer st.mu.Unlock()
	nodes := st.visibleNodesLocked()
	if st.selected < 0 || st.selected >= len(nodes) {
		return nil
	}
	return nodes[st.selected]
}

// Invalidate marks the component for re-render
func (st *SessionTree) Invalidate() {
	st.mu.Lock()
	st.dirty = true
	st.mu.Unlock()
}

// HandleKey processes a parsed key event for navigation
func (st *SessionTree) HandleKey(k key.Key) {
	st.mu.Lock()
	switch k.Type {
	case key.KeyUp:
		st.moveUpLocked()
	case key.KeyDown:
		st.moveDownLocked()
	case key.KeyEnter:
		// Accept selection: handled by caller
	case key.KeyEscape:
		st.filter = ""
		st.selected = 0
		st.scrollOff = 0
		st.dirty = true
	}
	st.mu.Unlock()
}

func (st *SessionTree) moveUp() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.moveUpLocked()
}

func (st *SessionTree) moveUpLocked() {
	if st.selected > 0 {
		st.selected--
		st.adjustScrollLocked()
		st.dirty = true
	}
}

func (st *SessionTree) moveDown() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.moveDownLocked()
}

func (st *SessionTree) moveDownLocked() {
	nodes := st.visibleNodesLocked()
	if st.selected < len(nodes)-1 {
		st.selected++
		st.adjustScrollLocked()
		st.dirty = true
	}
}

func (st *SessionTree) adjustScrollLocked() {
	if st.selected < st.scrollOff {
		st.scrollOff = st.selected
	}
	if st.selected >= st.scrollOff+st.maxHeight {
		st.scrollOff = st.selected - st.maxHeight + 1
	}
}

func (st *SessionTree) visibleNodesLocked() []*SessionNode {
	var nodes []*SessionNode
	for _, root := range st.roots {
		st.collectVisibleNodes(root, &nodes, 0)
	}
	return nodes
}

func (st *SessionTree) collectVisibleNodes(node *SessionNode, nodes *[]*SessionNode, indent int) {
	// Check if node matches filter
	if st.filterMatches(node) {
		node.Level = indent
		node.IsBranch = len(node.Children) > 0
		*nodes = append(*nodes, node)
	}

	// Recurse into children
	for _, child := range node.Children {
		st.collectVisibleNodes(child, nodes, indent+1)
	}
}

func (st *SessionTree) filterMatches(node *SessionNode) bool {
	if st.filter == "" {
		return true
	}

	// Check ID and model for fuzzy match
	if fuzzy.Find(st.filter, []string{node.ID, node.Model}) != nil {
		return true
	}

	return false
}

// LoadSessionsFromDir loads all sessions from a directory and builds the tree
func LoadSessionsFromDir(dir string) ([]*SessionNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var sessionIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			sessionIDs = append(sessionIDs, strings.TrimSuffix(entry.Name(), ".jsonl"))
		}
	}

	// Read all session start records
	var nodes []*SessionNode
	for _, id := range sessionIDs {
		records, err := session.ReadRecords(id)
		if err != nil {
			continue
		}

		// Find session_start record
		var start *session.SessionStartData
		for _, rec := range records {
			if rec.Type == session.RecordSessionStart && rec.Data != nil {
				var data session.SessionStartData
				if err := json.Unmarshal(rec.Data, &data); err == nil {
					start = &data
					break
				}
			}
		}

		if start == nil {
			continue
		}

		node := &SessionNode{
			ID:       start.ID,
			Model:    start.Model,
			Start:    start,
			Count:    len(records),
			Children: nil,
		}
		nodes = append(nodes, node)
	}

	// Sort by ID (most recent first - session IDs are time-based)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID > nodes[j].ID
	})

	return nodes, nil
}

// Render writes the session tree into the buffer.
// Snapshots all state under lock to avoid data races with SetFilter/HandleKey.
func (st *SessionTree) Render(out *tui.RenderBuffer, w int) {
	st.mu.Lock()
	nodes := st.visibleNodesLocked()
	filter := st.filter
	scrollOff := st.scrollOff
	maxHeight := st.maxHeight
	selected := st.selected
	st.mu.Unlock()

	if len(nodes) == 0 {
		if filter == "" {
			out.WriteLine("\x1b[2mNo sessions found\x1b[0m")
		} else {
			out.WriteLine("\x1b[2mNo matching sessions\x1b[0m")
		}
		return
	}

	end := min(scrollOff+maxHeight, len(nodes))

	for i := scrollOff; i < end; i++ {
		node := nodes[i]
		line := st.formatNode(node, w, i == selected)
		out.WriteLine(line)
	}
}

func (st *SessionTree) formatNode(node *SessionNode, w int, selected bool) string {
	var line string

	// Build tree structure with indentation
	indent := strings.Repeat("  ", node.Level)

	if node.IsBranch {
		line = fmt.Sprintf("%s├── %s", indent, node.ID[:8])
	} else {
		line = fmt.Sprintf("%s└── %s", indent, node.ID[:8])
	}

	// Add model
	if node.Model != "" {
		line += fmt.Sprintf(" \x1b[36m%s\x1b[0m", node.Model)
	}

	// Add message count
	if node.Count > 0 {
		line += fmt.Sprintf(" \x1b[90m(%d)\x1b[0m", node.Count)
	}

	// Truncate to width
	line = strings.TrimSpace(line)
	if len(line) > w {
		line = line[:w-1] + "…"
	}

	if selected {
		line = "\x1b[1m\x1b[7m" + line + "\x1b[0m"
	}

	return line
}

// Count returns the number of visible nodes
func (st *SessionTree) Count() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.visibleNodesLocked())
}

// Reset clears the filter and selection
func (st *SessionTree) Reset() {
	st.mu.Lock()
	st.filter = ""
	st.selected = 0
	st.scrollOff = 0
	st.dirty = true
	st.mu.Unlock()
}

