// ABOUTME: Non-interactive print mode for piped/scripted output
// ABOUTME: Reads prompt from stdin or args, streams response to stdout

package print

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Run executes the agent in non-interactive mode.
func Run(ctx context.Context, provider ai.ApiProvider, model *ai.Model, prompt string) error {
	if prompt == "" {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		prompt = string(data)
	}

	llmCtx := &ai.Context{
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, prompt),
		},
	}

	stream := provider.Stream(model, llmCtx, &ai.StreamOptions{})
	_ = ctx // Context propagation handled by provider

	for event := range stream.Events() {
		switch event.Type {
		case ai.EventContentDelta:
			fmt.Print(event.Text)
		case ai.EventError:
			return fmt.Errorf("stream error: %w", event.Error)
		}
	}

	fmt.Println() // Final newline
	return nil
}
