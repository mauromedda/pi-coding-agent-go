// ABOUTME: Accumulates content blocks and metadata during Anthropic SSE streaming
// ABOUTME: Tracks current block state and builds the final AssistantMessage

package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// accumulator gathers streaming data into a final AssistantMessage.
type accumulator struct {
	model      string
	stopReason ai.StopReason
	usage      ai.Usage
	content    []ai.Content
	current    *blockState
}

// blockState tracks the in-progress content block.
type blockState struct {
	contentType ai.ContentType
	id          string
	name        string
	text        strings.Builder
	toolInput   strings.Builder
}

// newAccumulator creates an empty accumulator.
func newAccumulator() *accumulator {
	return &accumulator{}
}

// startBlock begins accumulating a new content block.
func (a *accumulator) startBlock(typeName, id, name string) {
	a.current = &blockState{
		contentType: ai.ContentType(typeName),
		id:          id,
		name:        name,
	}
}

// appendText adds text to the current block.
func (a *accumulator) appendText(text string) {
	if a.current != nil {
		a.current.text.WriteString(text)
	}
}

// appendToolInput adds partial JSON to the current tool use block.
func (a *accumulator) appendToolInput(partial string) {
	if a.current != nil {
		a.current.toolInput.WriteString(partial)
	}
}

// finishBlock finalizes the current block and appends it to content.
// Returns the finalized Content or nil if no block was in progress.
func (a *accumulator) finishBlock() *ai.Content {
	if a.current == nil {
		return nil
	}

	block := ai.Content{Type: a.current.contentType}

	switch a.current.contentType {
	case ai.ContentText:
		block.Text = a.current.text.String()
	case ai.ContentToolUse:
		block.ID = a.current.id
		block.Name = a.current.name
		block.Input = json.RawMessage(a.current.toolInput.String())
	}

	a.content = append(a.content, block)
	a.current = nil

	return &a.content[len(a.content)-1]
}

// buildResult constructs the final AssistantMessage from accumulated data.
func (a *accumulator) buildResult() *ai.AssistantMessage {
	return &ai.AssistantMessage{
		Content:    a.content,
		StopReason: a.stopReason,
		Usage:      a.usage,
		Model:      a.model,
	}
}
