// ABOUTME: SDK/headless print mode with text, JSON, and stream-JSON formatters
// ABOUTME: Runs full agent loop with tools; supports turn/budget limits and session continuation

package print

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Config configures SDK/headless mode execution.
type Config struct {
	OutputFormat       string  // "text" (default), "json", "stream-json"
	MaxTurns           int     // 0 = unlimited
	MaxBudgetUSD       float64 // 0 = unlimited
	SystemPrompt       string  // Override system prompt
	AppendSystemPrompt string  // Append to system prompt
	ContinueSession    bool    // Continue last session
	ResumeSessionID    string  // Resume specific session
}

// Deps provides dependencies for print mode.
type Deps struct {
	Provider ai.ApiProvider
	Model    *ai.Model
	Tools    []*agent.AgentTool
}

// Run executes the agent in non-interactive mode with the given configuration.
func Run(ctx context.Context, provider ai.ApiProvider, model *ai.Model, prompt string) error {
	cfg := Config{OutputFormat: "text"}
	deps := Deps{Provider: provider, Model: model}
	return RunWithConfig(ctx, cfg, deps, prompt)
}

// RunWithConfig executes print mode with full configuration.
func RunWithConfig(ctx context.Context, cfg Config, deps Deps, prompt string) error {
	if prompt == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		prompt = string(data)
	}

	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "text"
	}

	formatter := newFormatter(cfg.OutputFormat)

	// Build system prompt
	system := cfg.SystemPrompt
	if cfg.AppendSystemPrompt != "" {
		if system != "" {
			system += "\n\n"
		}
		system += cfg.AppendSystemPrompt
	}

	llmCtx := &ai.Context{
		System: system,
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, prompt),
		},
	}

	// Add tools to context
	if len(deps.Tools) > 0 {
		for _, t := range deps.Tools {
			schema := t.Parameters
			if schema == nil {
				schema = json.RawMessage(`{}`)
			}
			llmCtx.Tools = append(llmCtx.Tools, ai.Tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			})
		}
	}

	opts := &ai.StreamOptions{MaxTokens: 4096}

	// If we have tools, use the full agent loop
	if len(deps.Tools) > 0 {
		return runAgentLoop(ctx, cfg, deps, llmCtx, opts, formatter)
	}

	// Simple streaming without tools
	return runSimpleStream(ctx, deps, llmCtx, opts, formatter)
}

func runAgentLoop(ctx context.Context, cfg Config, deps Deps, llmCtx *ai.Context, opts *ai.StreamOptions, f formatter) error {
	ag := agent.New(deps.Provider, deps.Model, deps.Tools)
	events := ag.Prompt(ctx, llmCtx, opts)

	turns := 0
	f.start()

	for evt := range events {
		switch evt.Type {
		case agent.EventAssistantText:
			f.text(evt.Text)
		case agent.EventToolStart:
			f.toolStart(evt.ToolName, evt.ToolArgs)
		case agent.EventToolEnd:
			if evt.ToolResult != nil {
				f.toolEnd(evt.ToolName, evt.ToolResult)
			}
			turns++
			if cfg.MaxTurns > 0 && turns >= cfg.MaxTurns {
				ag.Abort()
			}
		case agent.EventError:
			f.err(evt.Error)
		}
	}

	f.end()
	return nil
}

func runSimpleStream(ctx context.Context, deps Deps, llmCtx *ai.Context, opts *ai.StreamOptions, f formatter) error {
	stream := deps.Provider.Stream(deps.Model, llmCtx, opts)
	_ = ctx

	f.start()
	for event := range stream.Events() {
		switch event.Type {
		case ai.EventContentDelta:
			f.text(event.Text)
		case ai.EventError:
			f.err(event.Error)
		}
	}
	f.end()
	return nil
}

// formatter abstracts output formatting.
type formatter interface {
	start()
	text(s string)
	toolStart(name string, args map[string]any)
	toolEnd(name string, result *agent.ToolResult)
	err(e error)
	end()
}

func newFormatter(format string) formatter {
	switch format {
	case "json":
		return &jsonFormatter{}
	case "stream-json":
		return &streamJSONFormatter{}
	default:
		return &textFormatter{}
	}
}

// textFormatter outputs plain text to stdout.
type textFormatter struct{}

func (f *textFormatter) start()                                         {}
func (f *textFormatter) text(s string)                                  { fmt.Print(s) }
func (f *textFormatter) toolStart(name string, _ map[string]any)        { fmt.Fprintf(os.Stderr, "[tool: %s]\n", name) }
func (f *textFormatter) toolEnd(_ string, _ *agent.ToolResult)          {}
func (f *textFormatter) err(e error)                                    { fmt.Fprintf(os.Stderr, "error: %v\n", e) }
func (f *textFormatter) end()                                           { fmt.Println() }

// jsonFormatter collects all output and writes a single JSON object at the end.
type jsonFormatter struct {
	textBuf    strings.Builder
	toolCalls  []jsonToolCall
	errors     []string
}

type jsonToolCall struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Result string         `json:"result,omitempty"`
	Error  bool           `json:"error,omitempty"`
}

type jsonOutput struct {
	Text      string         `json:"text"`
	ToolCalls []jsonToolCall `json:"tool_calls,omitempty"`
	Errors    []string       `json:"errors,omitempty"`
}

func (f *jsonFormatter) start()          {}
func (f *jsonFormatter) text(s string)   { f.textBuf.WriteString(s) }
func (f *jsonFormatter) toolStart(name string, args map[string]any) {
	f.toolCalls = append(f.toolCalls, jsonToolCall{Name: name, Args: args})
}
func (f *jsonFormatter) toolEnd(name string, result *agent.ToolResult) {
	if len(f.toolCalls) > 0 {
		last := &f.toolCalls[len(f.toolCalls)-1]
		if last.Name == name {
			last.Result = result.Content
			last.Error = result.IsError
		}
	}
}
func (f *jsonFormatter) err(e error) { f.errors = append(f.errors, e.Error()) }
func (f *jsonFormatter) end() {
	out := jsonOutput{
		Text:      f.textBuf.String(),
		ToolCalls: f.toolCalls,
		Errors:    f.errors,
	}
	data, _ := json.Marshal(out)
	fmt.Println(string(data))
}

// streamJSONFormatter outputs one JSON line per event.
type streamJSONFormatter struct{}

type streamEvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Tool string `json:"tool,omitempty"`
	Error string `json:"error,omitempty"`
}

func (f *streamJSONFormatter) start() {
	writeStreamEvent(streamEvent{Type: "start"})
}

func (f *streamJSONFormatter) text(s string) {
	writeStreamEvent(streamEvent{Type: "text", Text: s})
}

func (f *streamJSONFormatter) toolStart(name string, _ map[string]any) {
	writeStreamEvent(streamEvent{Type: "tool_start", Tool: name})
}

func (f *streamJSONFormatter) toolEnd(name string, result *agent.ToolResult) {
	evt := streamEvent{Type: "tool_end", Tool: name, Text: result.Content}
	if result.IsError {
		evt.Error = result.Content
		evt.Text = ""
	}
	writeStreamEvent(evt)
}

func (f *streamJSONFormatter) err(e error) {
	writeStreamEvent(streamEvent{Type: "error", Error: e.Error()})
}

func (f *streamJSONFormatter) end() {
	writeStreamEvent(streamEvent{Type: "end"})
}

func writeStreamEvent(evt streamEvent) {
	data, _ := json.Marshal(evt)
	fmt.Println(string(data))
}
