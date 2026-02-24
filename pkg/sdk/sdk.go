// ABOUTME: Public SDK for programmatic use of the pi-coding-agent
// ABOUTME: Wraps internal agent with functional options and convenience result types

package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// defaultModelID is the fallback model when none is specified.
const defaultModelID = "claude-sonnet-4-6"

// Client is the main entry point for the SDK.
type Client struct {
	provider     ai.ApiProvider
	model        *ai.Model
	tools        []*agent.AgentTool
	systemPrompt string
	maxTurns     int
	apiKey       string
	ctx          context.Context
	cancel       context.CancelFunc
	handlers     []func(agent.AgentEvent)
}

// Option configures a Client.
type Option func(*clientConfig)

type clientConfig struct {
	modelID      string
	model        *ai.Model
	provider     ai.ApiProvider
	apiKey       string
	systemPrompt string
	maxTurns     int
	tools        []*agent.AgentTool
}

// WithModel sets the model by ID (from the built-in catalog).
func WithModel(id string) Option {
	return func(c *clientConfig) {
		c.modelID = id
	}
}

// WithModelDirect sets the model directly (bypasses catalog lookup).
func WithModelDirect(m *ai.Model) Option {
	return func(c *clientConfig) {
		c.model = m
	}
}

// WithAPIKey sets the API key for the LLM provider.
func WithAPIKey(key string) Option {
	return func(c *clientConfig) {
		c.apiKey = key
	}
}

// WithSystemPrompt sets the system prompt for the agent.
func WithSystemPrompt(prompt string) Option {
	return func(c *clientConfig) {
		c.systemPrompt = prompt
	}
}

// WithMaxTurns sets the maximum number of agent turns per prompt.
func WithMaxTurns(n int) Option {
	return func(c *clientConfig) {
		c.maxTurns = n
	}
}

// WithTool registers an additional tool with the agent.
func WithTool(t *agent.AgentTool) Option {
	return func(c *clientConfig) {
		c.tools = append(c.tools, t)
	}
}

// WithProvider sets the LLM provider directly (for testing or custom providers).
func WithProvider(p ai.ApiProvider) Option {
	return func(c *clientConfig) {
		c.provider = p
	}
}

// New creates a new SDK client with the given options.
func New(opts ...Option) (*Client, error) {
	cfg := &clientConfig{}
	for _, o := range opts {
		o(cfg)
	}

	// Resolve model
	var model *ai.Model
	if cfg.model != nil {
		model = cfg.model
	} else {
		modelID := cfg.modelID
		if modelID == "" {
			modelID = defaultModelID
		}
		model = ai.FindModel(modelID)
		if model == nil {
			return nil, fmt.Errorf("model %q not found in built-in catalog", modelID)
		}
	}

	// Resolve provider
	if cfg.provider == nil {
		provider := ai.GetProvider(model.Api, model.BaseURL)
		if provider == nil {
			return nil, fmt.Errorf("no provider registered for API %q; use WithProvider to supply one", model.Api)
		}
		cfg.provider = provider
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		provider:     cfg.provider,
		model:        model,
		tools:        cfg.tools,
		systemPrompt: cfg.systemPrompt,
		maxTurns:     cfg.maxTurns,
		apiKey:       cfg.apiKey,
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}

// Prompt sends a user message and returns the agent response.
// The prompt is cancelled if either the caller's ctx or the client's internal
// context (via Close) is done.
func (c *Client) Prompt(ctx context.Context, text string) (*Result, error) {
	// Derive from the client's lifecycle context so Close() cancels in-flight work.
	promptCtx, promptCancel := context.WithCancel(c.ctx)
	defer promptCancel()

	// Also cancel if the caller's context is done.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	stop := context.AfterFunc(ctx, promptCancel)
	defer stop()

	ag := agent.New(c.provider, c.model, c.tools)

	llmCtx := &ai.Context{
		System: c.systemPrompt,
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, text),
		},
	}

	eventCh := ag.Prompt(promptCtx, llmCtx, &ai.StreamOptions{})

	var messages []ai.Message
	for evt := range eventCh {
		// Forward events to registered handlers
		for _, h := range c.handlers {
			h(evt)
		}

		// Collect assistant text into messages
		switch evt.Type {
		case agent.EventAssistantText:
			// Events are streamed; we'll get the full message from the final state
		case agent.EventError:
			if evt.Error != nil {
				return nil, evt.Error
			}
		}
	}

	// Build result from the LLM context (which now includes assistant + tool messages)
	// The agent appends to llmCtx.Messages as it runs
	for _, msg := range llmCtx.Messages[1:] { // skip the initial user message
		messages = append(messages, msg)
	}

	return &Result{Messages: messages}, nil
}

// OnEvent registers a listener for agent lifecycle events.
func (c *Client) OnEvent(handler func(agent.AgentEvent)) {
	c.handlers = append(c.handlers, handler)
}

// Close cleans up the client resources.
func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// Result wraps the agent result with convenience methods.
type Result struct {
	Messages []ai.Message
}

// Text returns the concatenated text content from assistant messages.
func (r *Result) Text() string {
	var parts []string
	for _, m := range r.Messages {
		if m.Role != ai.RoleAssistant {
			continue
		}
		for _, c := range m.Content {
			if c.Type == ai.ContentText && c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
	}
	return strings.Join(parts, "")
}

// ToolCalls returns all tool_use content blocks from assistant messages.
func (r *Result) ToolCalls() []ai.Content {
	var calls []ai.Content
	for _, m := range r.Messages {
		if m.Role != ai.RoleAssistant {
			continue
		}
		for _, c := range m.Content {
			if c.Type == ai.ContentToolUse {
				calls = append(calls, c)
			}
		}
	}
	return calls
}
